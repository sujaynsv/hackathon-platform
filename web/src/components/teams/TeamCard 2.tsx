'use client';

import React, { useState } from 'react';
import type { Team } from '../../types/api';

interface TeamCardProps {
  team: Team;
  onLeave: () => void;
  currentUserId?: string;
}

export function TeamCard({ team, onLeave, currentUserId }: TeamCardProps) {
  const [showConfirm, setShowConfirm] = useState(false);
  const [confirmText, setConfirmText] = useState('');
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(team.inviteCode);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const isLeader = team.members.find((m) => m.userId === currentUserId)?.role === 'owner';

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 shadow-xl relative overflow-hidden">
      <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-blue-500 to-teal-400" />
      
      <div className="flex justify-between items-start mb-6">
        <div>
          <h2 className="text-2xl font-bold text-white mb-2">{team.name}</h2>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <span>Invite Code:</span>
            <div className="flex items-center bg-gray-800 rounded px-2 py-1 gap-2">
              <code className="font-mono text-teal-400">{team.inviteCode}</code>
              <button
                onClick={handleCopy}
                className="text-gray-400 hover:text-white transition-colors"
                title="Copy invite code"
              >
                {copied ? (
                  <span className="text-xs text-green-400">Copied!</span>
                ) : (
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-4 mb-8">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wider">Members ({team.members.length})</h3>
        <div className="grid gap-3">
          {team.members.map((member) => (
            <div key={member.id} className="flex items-center justify-between bg-gray-800/50 p-3 rounded-lg border border-gray-800">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-full bg-gray-700 flex items-center justify-center text-sm font-bold text-gray-300">
                  {/* Fallback to initials since we don't have user profiles hydrated yet in the DTO */}
                  U
                </div>
                <span className="text-gray-200">
                  User {member.userId.substring(0, 8)} {member.userId === currentUserId && "(You)"}
                </span>
              </div>
              {member.role === 'owner' ? (
                <span className="text-xs px-2 py-1 bg-blue-500/20 text-blue-400 rounded border border-blue-500/30">Leader</span>
              ) : (
                <span className="text-xs px-2 py-1 bg-gray-700 text-gray-400 rounded">Member</span>
              )}
            </div>
          ))}
        </div>
      </div>

      {!isLeader && (
        <div className="border-t border-gray-800 pt-6">
          {!showConfirm ? (
            <button
              onClick={() => setShowConfirm(true)}
              className="text-red-400 hover:text-red-300 text-sm font-medium transition-colors"
            >
              Leave Team
            </button>
          ) : (
            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4">
              <p className="text-sm text-red-200 mb-3">
                Are you sure you want to leave this team? Type <span className="font-bold">LEAVE</span> to confirm.
              </p>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={confirmText}
                  onChange={(e) => setConfirmText(e.target.value)}
                  placeholder="LEAVE"
                  className="bg-gray-900 border border-gray-700 text-white rounded px-3 py-1.5 text-sm flex-1 outline-none focus:border-red-500"
                />
                <button
                  onClick={onLeave}
                  disabled={confirmText !== 'LEAVE'}
                  className="bg-red-500 text-white px-4 py-1.5 rounded text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-red-600 transition-colors"
                >
                  Confirm
                </button>
                <button
                  onClick={() => setShowConfirm(false)}
                  className="bg-gray-700 text-white px-4 py-1.5 rounded text-sm font-medium hover:bg-gray-600 transition-colors"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
