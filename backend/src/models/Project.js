const mongoose = require('mongoose');

const projectSchema = new mongoose.Schema({
  title: { type: String, required: true },
  description: { type: String, required: true },
  category: {
    type: String,
    enum: ['web', 'mobile', 'design', 'marketing', 'other'],
    required: true
  },
  budget: { type: Number, required: true },
  duration: { type: Number, required: true },
  complexity: { type: Number, min: 1, max: 10, required: true },
  teamSize: { type: Number, required: true },
  kpis: [{
    name: { type: String, required: true },
    target: { type: String, required: true },
    weight: { type: Number, required: true }
  }],
  creator: { type: mongoose.Schema.Types.ObjectId, ref: 'User', required: true },
  successProbability: { type: Number, default: 0 },
  status: {
    type: String,
    enum: ['draft', 'active', 'completed', 'failed'],
    default: 'draft'
  }
}, { timestamps: true });

module.exports = mongoose.model('Project', projectSchema);