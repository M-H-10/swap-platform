// Logistic Regression Model لحساب احتمالية النجاح
function calculateSuccessProbability(projectData) {
  const {
    budget,
    duration,
    complexity,
    teamSize,
    category,
    creatorReputation,
    creatorSuccessRate,
    creatorCompletedProjects
  } = projectData;

  // معاملات النموذج (مبنية على بيانات تاريخية مُحاكاة)
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
    + weights.budget * Math.min(budget, 1000000)
    + weights.duration * duration
    + weights.complexity * complexity
    + weights.teamSize * Math.min(teamSize, 10)
    + weights.reputation * creatorReputation
    + weights.successRate * creatorSuccessRate
    + weights.completedProjects * Math.min(creatorCompletedProjects, 20)
    + (weights.categoryBonus[category] || 0);

  // Sigmoid function
  const probability = 1 / (1 + Math.exp(-z));
  return Math.max(0.05, Math.min(0.95, probability));
}

// حساب هيكل الـ Swap العادل
function calculateFairSwap(projectData, partyAData, partyBData) {
  const successProb = calculateSuccessProbability(projectData);
  const failProb = 1 - successProb;

  const { budget } = projectData;

  // حساب نسبة المساهمة بناءً على السمعة والخبرة
  const totalReputation = partyAData.reputation + partyBData.reputation;
  const shareA = partyAData.reputation / totalReputation;
  const shareB = partyBData.reputation / totalReputation;

  // تعديل الحصص بناءً على معدل النجاح
  const adjustedShareA = (shareA * 0.7) + (partyAData.successRate / 100 * 0.3);
  const adjustedShareB = 1 - adjustedShareA;

  // حساب المدفوعات الثابتة عند الفشل
  const escrowAmount = budget * 0.1;
  const fixedPaymentA = escrowAmount * adjustedShareB;
  const fixedPaymentB = escrowAmount * adjustedShareA;

  // حساب ملف المخاطر
  const riskProfileA = {
    expectedValue: (budget * adjustedShareA * successProb) - (fixedPaymentA * failProb),
    maxGain: budget * adjustedShareA,
    maxLoss: fixedPaymentA,
    riskScore: Math.round((failProb * fixedPaymentA / (budget * adjustedShareA)) * 100)
  };

  const riskProfileB = {
    expectedValue: (budget * adjustedShareB * successProb) - (fixedPaymentB * failProb),
    maxGain: budget * adjustedShareB,
    maxLoss: fixedPaymentB,
    riskScore: Math.round((failProb * fixedPaymentB / (budget * adjustedShareB)) * 100)
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