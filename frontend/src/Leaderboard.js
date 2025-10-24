import React, { useState, useEffect } from 'react';
import './Leaderboard.css';

function Leaderboard({ apiUrl, onBack }) {
  const [leaderboard, setLeaderboard] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchLeaderboard();
  }, []);

  const fetchLeaderboard = async () => {
    try {
      console.log('Fetching leaderboard from:', `${apiUrl}/leaderboard`);
      const response = await fetch(`${apiUrl}/leaderboard`);
      
      if (!response.ok) {
        throw new Error(`Failed to fetch leaderboard (Status: ${response.status})`);
      }
      
      const data = await response.json();
      console.log('Leaderboard data:', data);
      setLeaderboard(data || []);
      setLoading(false);
    } catch (err) {
      console.error('Leaderboard error:', err);
      setError(err.message);
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="container">
        <h1>🏆 Leaderboard</h1>
        <p>Loading...</p>
        <button onClick={onBack}>Back</button>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container">
        <h1>🏆 Leaderboard</h1>
        <p className="error">Error: {error}</p>
        <p>Please make sure the backend server is running on port 8080.</p>
        <div style={{ display: 'flex', gap: '10px', justifyContent: 'center', marginTop: '20px' }}>
          <button onClick={fetchLeaderboard}>Retry</button>
          <button onClick={onBack}>Back</button>
        </div>
      </div>
    );
  }

  return (
    <div className="container">
      <h1>🏆 Leaderboard</h1>
      {leaderboard.length === 0 ? (
        <p>No games played yet. Be the first!</p>
      ) : (
        <table className="leaderboard-table">
          <thead>
            <tr>
              <th>Rank</th>
              <th>Player</th>
              <th>Won</th>
              <th>Lost</th>
              <th>Drawn</th>
              <th>Played</th>
              <th>Win Rate</th>
            </tr>
          </thead>
          <tbody>
            {leaderboard.map((entry, index) => {
              const winRate = entry.gamesPlayed > 0
                ? ((entry.gamesWon / entry.gamesPlayed) * 100).toFixed(1)
                : '0.0';
              return (
                <tr key={entry.username}>
                  <td className="rank">{index + 1}</td>
                  <td className="player-name">{entry.username}</td>
                  <td className="wins">{entry.gamesWon}</td>
                  <td className="losses">{entry.gamesLost}</td>
                  <td className="draws">{entry.gamesDrawn}</td>
                  <td>{entry.gamesPlayed}</td>
                  <td className="win-rate">{winRate}%</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
      <div style={{ display: 'flex', gap: '10px', justifyContent: 'center', marginTop: '20px' }}>
        <button onClick={fetchLeaderboard}>Refresh</button>
        <button onClick={onBack}>Back</button>
      </div>
    </div>
  );
}

export default Leaderboard;
