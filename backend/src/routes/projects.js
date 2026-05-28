const express = require('express');
const router = express.Router();
const Project = require('../models/Project');
const User = require('../models/User');
const auth = require('../middleware/auth');
const { calculateSuccessProbability } = require('../utils/swapCalculator');

router.get('/', async (req, res) => {
  try {
    const projects = await Project.find({ status: 'draft' })
      .populate('creator', 'username reputation role successRate')
      .sort({ createdAt: -1 });
    res.json(projects);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
});

router.post('/', auth, async (req, res) => {
  try {
    const creator = await User.findById(req.user.userId);
    const projectData = {
      ...req.body,
      creatorReputation: creator.reputation,
      creatorSuccessRate: creator.successRate,
      creatorCompletedProjects: creator.completedProjects
    };
    const successProbability = calculateSuccessProbability(projectData);
    const project = new Project({
      ...req.body,
      creator: req.user.userId,
      successProbability: Math.round(successProbability * 100)
    });
    await project.save();
    res.status(201).json(project);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
});

router.get('/:id', async (req, res) => {
  try {
    const project = await Project.findById(req.params.id)
      .populate('creator', 'username reputation role successRate completedProjects');
    if (!project) return res.status(404).json({ message: 'Project not found' });
    res.json(project);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
});

module.exports = router;