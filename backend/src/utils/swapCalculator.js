// Logistic Regression Model
function calculateSuccessProbability(projectData) {
  const {
    budget = 0,
    duration = 0,
    complexity = 5,
    teamSize = 2,
    category = 'other',
    creatorReputation = 50,
    creatorSuccessRate = 0,
    creatorCompletedProjects = 0
  } = projectData;

  const weights = {
    budget: 0.000002,
    duration: -0.01,
    complexity: -0.08,
    teamSize: 0.05,
    reputation: 0.015,
    successRate: 0.008,
    completedProjects: 0.01,
    categoryBonus: {
      web: 0.1,
      mobile: 0.05,
      design: 0.15,
      marketing: 0.08,
      other: 0
    }
  };

  const bias = -0.5;

  const z = bias
    + weights.budget * Math.min(Math.max(budget, 0), 1000000)
    + weights.duration * Math.max(duration, 0)
    + weights.complexity * Math.min(Math.max(complexity, 1), 10)
    + weights.teamSize * Math.min(Math.max(teamSize, 1), 10)
    + weights.reputation * Math.min(Math.max(creatorReputation, 0), 100)
    + weights.successRate * Math.min(Math.max(creatorSuccessRate, 0), 100)
    + weights.completedProjects * Math.min(Math.max(creatorCompletedProjects, 0), 20)
    + (weights.categoryBonus[category] || 0);

  const probability = 1 / (1 + Math.exp(-z));
  return Math.max(0.05, Math.min(0.95, probability));
}

function calculateFairSwap(projectData, partyAData, partyBData) {
  const successProb = calculateSuccessProbability(projectData);
  const failProb = 1 - successProb;

  const budget = Math.max(projectData.budget || 0, 1);

  const totalReputation = (partyAData.reputation || 50) + (partyBData.reputation || 50);
  const shareA = totalReputation > 0 ? (partyAData.reputation || 50) / totalReputation : 0.5;

  // تعديل الحصص: 70% السمعة + 30% تاريخ النجاح
  const adjustedShareA = Math.min(Math.max(
    (shareA * 0.7) + ((partyAData.successRate || 0) / 100 * 0.3),
    0.05
  ), 0.95);
  const adjustedShareB = 1 - adjustedShareA;

  const escrowAmount = budget * 0.1;
  const fixedPaymentA = escrowAmount * adjustedShareB;
  const fixedPaymentB = escrowAmount * adjustedShareA;

  // حساب riskScore مع حماية من القسمة على صفر
  const riskScoreA = (budget * adjustedShareA) > 0
    ? Math.min(Math.round((failProb * fixedPaymentA / (budget * adjustedShareA)) * 100), 100)
    : 0;
  const riskScoreB = (budget * adjustedShareB) > 0
    ? Math.min(Math.round((failProb * fixedPaymentB / (budget * adjustedShareB)) * 100), 100)
    : 0;

  const riskProfileA = {
    expectedValue: Math.round((budget * adjustedShareA * successProb) - (fixedPaymentA * failProb)),
    maxGain: Math.round(budget * adjustedShareA),
    maxLoss: Math.round(fixedPaymentA),
    riskScore: riskScoreA
  };

  const riskProfileB = {
    expectedValue: Math.round((budget * adjustedShareB * successProb) - (fixedPaymentB * failProb)),
    maxGain: Math.round(budget * adjustedShareB),
    maxLoss: Math.round(fixedPaymentB),
    riskScore: riskScoreB
  };

  return {
    successProbability: Math.round(successProb * 100),
    swapRatio: Math.round(adjustedShareA * 100) / 100,
    partyA: {
      shareOnSuccess: Math.round(adjustedShareA * 100),
      fixedPaymentOnFailure: Math.round(fixedPaymentA),
      riskProfile: riskProfileA
    },
    partyB: {
      shareOnSuccess: Math.round(adjustedShareB * 100),
      fixedPaymentOnFailure: Math.round(fixedPaymentB),
      riskProfile: riskProfileB
    },
    escrowAmount: Math.round(escrowAmount)
  };
}

module.exports = { calculateSuccessProbability, calculateFairSwap };