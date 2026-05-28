const mongoose = require('mongoose');

const userSchema = new mongoose.Schema({
  username: { type: String, required: true, unique: true, trim: true },
  email: { type: String, required: true, unique: true, trim: true },
  password: { type: String, required: true },
  reputation: { type: Number, default: 50 },
  completedProjects: { type: Number, default: 0 },
  successRate: { type: Number, default: 0 },
  role: { type: String, enum: ['developer', 'designer', 'manager', 'other'], default: 'developer' }
}, { timestamps: true });

module.exports = mongoose.model('User', userSchema);