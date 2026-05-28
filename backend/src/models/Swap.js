const mongoose = require('mongoose');

const swapSchema = new mongoose.Schema({
  project: { type: mongoose.Schema.Types.ObjectId, ref: 'Project', required: true },
  partyA: {
    user: { type: mongoose.Schema.Types.ObjectId, ref: 'User', required: true },
    role: { type: String, required: true },
    shareOnSuccess: { type: Number, required: true },
    fixedPaymentOnFailure: { type: Number, required: true }
  },
  partyB: {
    user: { type: mongoose.Schema.Types.ObjectId, ref: 'User', required: true },
    role: { type: String, required: true },
    shareOnSuccess: { type: Number, required: true },
    fixedPaymentOnFailure: { type: Number, required: true }
  },
  escrowAmount: { type: Number, required: true },
  escrowStatus: {
    type: String,
    enum: ['pending', 'locked', 'released', 'returned'],
    default: 'pending'
  },
  swapRatio: { type: Number, required: true },
  riskProfileA: {
    expectedValue: Number,
    maxLoss: Number,
    maxGain: Number,
    riskScore: Number
  },
  riskProfileB: {
    expectedValue: Number,
    maxLoss: Number,
    maxGain: Number,
    riskScore: Number
  },
  status: {
    type: String,
    enum: ['proposed', 'accepted', 'active', 'completed', 'cancelled'],
    default: 'proposed'
  },
  acceptedAt: Date,
  completedAt: Date
}, { timestamps: true });

module.exports = mongoose.model('Swap', swapSchema);