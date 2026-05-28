const express = require('express');
const router = express.Router();
const Swap = require('../models/Swap');
const Project = require('../models/Project');
const User = require('../models/User');
const auth = require('../middleware/auth');
const { calculateFairSwap } = require('../utils/swapCalculator');

// حساب الـ Swap المقترح
router.post('/calculate', auth, async (req, res) => {
  try {
    const { projectId, partyBId } = req.body;
    const project = await Project.findById(projectId).populate('creator');
    const partyA = await User.findById(req.user.userId);
    const partyB = await User.findById(partyBId);
    if (!project || !partyA || !partyB) return res.status(404).json({ message: 'Not found' });
    const result = calculateFairSwap(
      { ...project.toObject(), budget: project.budget, duration: project.duration,
        complexity: project.complexity, teamSize: project.teamSize, category: project.category,
        creatorReputation: partyA.reputation, creatorSuccessRate: partyA.successRate,
        creatorCompletedProjects: partyA.completedProjects },
      { reputation: partyA.reputation, successRate: partyA.successRate },
      { reputation: partyB.reputation, successRate: partyB.successRate }
    );
    res.json(result);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
});

// إنشاء عقد Swap
router.post('/', auth, async (req, res) => {
  try {
    const { projectId, partyBId, partyARole, partyBRole } = req.body;
    const project = await Project.findById(projectId);
    const partyA = await User.findById(req.user.userId);
    const partyB = await User.findById(partyBId);
    const swapData = calculateFairSwap(
      { ...project.toObject(), creatorReputation: partyA.reputation,
        creatorSuccessRate: partyA.successRate, creatorCompletedProjects: partyA.completedProjects },
      { reputation: partyA.reputation, successRate: partyA.successRate },
      { reputation: partyB.reputation, successRate: partyB.successRate }
    );
    const swap = new Swap({
      project: projectId,
      partyA: {
        user: req.user.userId, role: partyARole,
        shareOnSuccess: swapData.partyA.shareOnSuccess,
        fixedPaymentOnFailure: swapData.partyA.fixedPaymentOnFailure
      },
      partyB: {
        user: partyBId, role: partyBRole,
        shareOnSuccess: swapData.partyB.shareOnSuccess,
        fixedPaymentOnFailure: swapData.partyB.fixedPaymentOnFailure
      },
      escrowAmount: swapData.escrowAmount,
      swapRatio: swapData.swapRatio,
      riskProfileA: swapData.partyA.riskProfile,
      riskProfileB: swapData.partyB.riskProfile,
    });
    await swap.save();
    res.status(201).json(swap);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
});

// جلب جميع العقود
router.get('/', auth, async (req, res) => {
  try {
    const swaps = await Swap.find({
      $or: [{ 'partyA.user': req.user.userId }, { 'partyB.user': req.user.userId }]
    }).populate('project').populate('partyA.user', 'username reputation')
      .populate('partyB.user', 'username reputation').sort({ createdAt: -1 });
    res.json(swaps);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
});

// قبول العقد وتفعيل Escrow
router.patch('/:id/accept', auth, async (req, res) => {
  try {
    const swap = await Swap.findById(req.params.id);
    if (!swap) return res.status(404).json({ message: 'Swap not found' });
    if (swap.partyB.user.toString() !== req.user.userId)
      return res.status(403).json({ message: 'Not authorized' });
    swap.status = 'active';
    swap.escrowStatus = 'locked';
    swap.acceptedAt = new Date();
    await swap.save();
    res.json(swap);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
});

module.exports = router;