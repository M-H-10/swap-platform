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
    const { budget, duration, complexity, teamSize, title, description, category } = req.body;

    // Validation
    if (!title || title.trim().length < 3) return res.status(400).json({ message: 'Title must be at least 3 characters' });
    if (!description || description.trim().length < 10) return res.status(400).json({ message: 'Description too short' });
    if (!budget || budget <= 0) return res.status(400).json({ message: 'Budget must be positive' });
    if (!duration || duration <= 0) return res.status(400).json({ message: 'Duration must be positive' });
    if (!complexity || complexity < 1 || complexity > 10) return res.status(400).json({ message: 'Complexity must be 1-10' });
    if (!teamSize || teamSize < 1) return res.status(400).json({ message: 'Team size must be at least 1' });
    
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