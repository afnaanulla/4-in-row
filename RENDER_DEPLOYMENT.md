# 🚀 Deploy to Render.com - Complete Guide

## Why Render.com?
- ✅ Free tier available
- ✅ Easy deployment from GitHub
- ✅ PostgreSQL included
- ✅ No credit card required for free tier
- ✅ Automatic HTTPS/SSL

**Note:** Kafka not included in free tier (use simple setup without analytics)

---

## 📋 Pre-Deployment Checklist

- [ ] GitHub account created
- [ ] Code pushed to GitHub repository
- [ ] Render.com account created (https://render.com)
- [ ] Docker files present in project

---

## 🎯 Step-by-Step Deployment

### Step 1: Push Code to GitHub

```powershell
# Initialize git (if not already)
git init

# Add all files
git add .

# Commit
git commit -m "Production ready 4-in-a-row game"

# Add remote (replace with your repo URL)
git remote add origin https://github.com/YOUR_USERNAME/4-in-a-row.git

# Push
git push -u origin main
```

**Important:** Make sure `.env` files are in `.gitignore` (already done)

---

### Step 2: Create Render Account

1. Go to https://render.com
2. Click "Get Started"
3. Sign up with GitHub (recommended)
4. Authorize Render to access your repositories

---

### Step 3: Create PostgreSQL Database

1. **Click "New +"** → **"PostgreSQL"**

2. **Configure database:**
   - Name: `fourinarow-db`
   - Database: `fourinarow`
   - User: `fourinarow` (or leave default)
   - Region: Choose closest to you
   - Plan: **Free**

3. **Click "Create Database"**

4. **Copy connection details:**
   - Internal Database URL (for backend)
   - External Database URL (for local testing)

**Save these! You'll need them later.**

---

### Step 4: Deploy Backend Service

1. **Click "New +"** → **"Web Service"**

2. **Connect repository:**
   - Select your GitHub repository
   - Repository: `4-in-a-row`

3. **Configure service:**
   - Name: `fourinarow-backend`
   - Region: Same as database
   - Branch: `main`
   - Root Directory: `backend`
   - Runtime: **Docker**
   - Instance Type: **Free**

4. **Environment Variables:**
   Click "Advanced" → "Add Environment Variable"

   Add these variables:
   ```
   DB_HOST=<your-postgres-host>
   DB_PORT=5432
   DB_USER=<your-postgres-user>
   DB_PASSWORD=<your-postgres-password>
   DB_NAME=fourinarow
   KAFKA_BROKER=
   KAFKA_TOPIC=game-events
   PORT=8080
   ```

   **Where to find database values:**
   - Go to your PostgreSQL service
   - Click "Info" tab
   - Use "Internal Database URL" format: `postgres://user:password@host:port/database`
   - Extract: host, user, password from the URL

5. **Click "Create Web Service"**

**Wait 5-10 minutes for deployment...**

---

### Step 5: Deploy Frontend

1. **Click "New +"** → **"Static Site"**

2. **Connect repository:**
   - Same repository: `4-in-a-row`

3. **Configure:**
   - Name: `fourinarow-frontend`
   - Branch: `main`
   - Root Directory: `frontend`
   - Build Command: `npm install && npm run build`
   - Publish Directory: `build`

4. **Environment Variables:**
   Add:
   ```
   REACT_APP_WS_URL=wss://fourinarow-backend.onrender.com/ws
   REACT_APP_API_URL=https://fourinarow-backend.onrender.com/api
   ```

   **Replace `fourinarow-backend` with your actual backend service name!**

5. **Click "Create Static Site"**

**Wait 3-5 minutes for build...**

---

## ✅ Verify Deployment

### Test Backend

1. Go to your backend URL: `https://fourinarow-backend.onrender.com`
2. Test health endpoint:
   ```
   https://yourinarow-backend.onrender.com/api/health
   ```
   Should return:
   ```json
   {
     "status": "ok",
     "database": "connected",
     "kafka": "disabled"
   }
   ```

### Test Frontend

1. Open your frontend URL: `https://fourinarow-frontend.onrender.com`
2. Enter username and join game
3. Wait for bot (10 seconds)
4. Play game!

---

## 🔧 Troubleshooting

### Backend Won't Start

**Check logs:**
1. Go to Render Dashboard
2. Click on backend service
3. Click "Logs" tab
4. Look for errors

**Common issues:**
- Database connection failed → Check DB credentials
- Port binding error → Ensure PORT=8080
- Build failed → Check Dockerfile syntax

### Frontend Can't Connect

**Fix:**
1. Go to Frontend service settings
2. Check environment variables:
   - `REACT_APP_WS_URL` should use `wss://` (secure WebSocket)
   - `REACT_APP_API_URL` should use `https://`
   - Both should point to your backend URL
3. Trigger redeploy: Manual Deploy → "Clear build cache & deploy"

### Database Connection Timeout

**Solution:**
- Use **Internal Database URL** (not External) for backend
- Ensure backend and database in same region
- Check database status (should be "Available")

### CORS Errors

**Backend is already configured** to allow all origins.
If issues persist:
1. Check browser console for specific error
2. Verify backend URL in frontend env vars
3. Ensure backend service is running

---

## 📊 Monitoring Your App

### Check Backend Logs
```
Render Dashboard → Backend Service → Logs
```

Look for:
- Database connected successfully
- Server starting on port 8080
- WebSocket connections

### Check Frontend Logs
```
Render Dashboard → Frontend → Deploy Logs
```

### View Metrics
- Render provides basic metrics (free tier)
- CPU, Memory, Request count
- Located in "Metrics" tab

---

## 💰 Cost & Limits (Free Tier)

### What's Included:
- ✅ 750 hours/month of runtime (backend)
- ✅ 100GB bandwidth/month
- ✅ PostgreSQL database (90 days, then sleeps)
- ✅ Automatic HTTPS
- ✅ Unlimited static sites

### Limitations:
- ⚠️ Services sleep after 15 min of inactivity
- ⚠️ Cold start: 30-60 seconds wake-up time
- ⚠️ No Kafka support on free tier
- ⚠️ Database resets after 90 days

### Upgrading:
- **Starter:** $7/month (no sleep, faster)
- **Standard:** $25/month (better performance)

---

## 🔐 Security Best Practices

### For Production:

1. **Change database password:**
   ```
   Render Dashboard → Database → Settings → Reset Database
   ```

2. **Add custom domain:**
   ```
   Frontend Settings → Custom Domain → Add
   ```

3. **Enable branch deploys:**
   ```
   Settings → Auto-Deploy → Enable
   ```

4. **Set up notifications:**
   ```
   Settings → Notifications → Email/Slack
   ```

---

## 🎨 Update Frontend URLs

After deployment, update in GitHub:

**File:** `frontend/.env.production`
```bash
REACT_APP_WS_URL=wss://yourapp-backend.onrender.com/ws
REACT_APP_API_URL=https://yourapp-backend.onrender.com/api
```

Commit and push → Auto-deploys!

---

## 📝 Post-Deployment Checklist

- [ ] Backend health check returns OK
- [ ] Frontend loads correctly
- [ ] Can enter username and join game
- [ ] Bot joins after 10 seconds
- [ ] Can make moves
- [ ] Leaderboard shows data
- [ ] Winner messages display correctly
- [ ] Stats show in leaderboard (wins/losses/draws)

---

## 🚀 Alternative: Railway.app

If Render doesn't work, try Railway:

1. Go to https://railway.app
2. Click "Start a New Project"
3. Select "Deploy from GitHub"
4. Choose your repository
5. Railway auto-detects and deploys!

**Pros:**
- Simpler setup
- Better free tier
- No sleep time

**Cons:**
- $5 credit/month limit (paid after that)

---

## 📧 Submission URLs

After deployment, you'll have:

**Frontend URL:**
```
https://fourinarow-frontend.onrender.com
```

**Backend URL:**
```
https://fourinarow-backend.onrender.com
```

**GitHub Repository:**
```
https://github.com/YOUR_USERNAME/4-in-a-row
```

**Use these for assignment submission!**

---

## 🆘 Getting Help

**Render Support:**
- Community: https://community.render.com
- Docs: https://render.com/docs
- Status: https://status.render.com

**Common Issues:**
1. Build fails → Check Dockerfile
2. Database error → Verify credentials
3. CORS error → Check frontend env vars
4. 502 error → Backend not responding (check logs)

---

## ✅ Success!

Once deployed:
- ✅ Your game is live on the internet
- ✅ Anyone can access it
- ✅ Automatic HTTPS enabled
- ✅ Database persistent
- ✅ Ready for submission

**Share your link and play! 🎮**

---

## 📊 Quick Reference

| Service | Type | Purpose | Free Tier |
|---------|------|---------|-----------|
| Backend | Web Service | Go server | ✅ Yes |
| Frontend | Static Site | React app | ✅ Yes |
| Database | PostgreSQL | Data storage | ✅ 90 days |
| Analytics | ❌ Skip | Kafka needs paid | ❌ No |

---

**Last Updated:** 2025-01-24  
**Status:** Ready for Deployment 🚀
