import axios from 'axios';

const API = axios.create({ baseURL: 'http://localhost:5001/api' });

API.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

export const register = (data) => API.post('/auth/register', data);
export const login = (data) => API.post('/auth/login', data);
export const getMe = () => API.get('/auth/me');
export const getProjects = () => API.get('/projects');
export const getProject = (id) => API.get(`/projects/${id}`);
export const createProject = (data) => API.post('/projects', data);
export const calculateSwap = (data) => API.post('/swaps/calculate', data);
export const createSwap = (data) => API.post('/swaps', data);
export const getSwaps = () => API.get('/swaps');
export const acceptSwap = (id) => API.patch(`/swaps/${id}/accept`);