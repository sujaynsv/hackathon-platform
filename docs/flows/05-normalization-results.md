# Flow 5 — Score Normalization & Results Publication

> Covers the full normalization pipeline and results publication.
> **Invariants enforced**: I8, I16

---

## End-to-End Flow Diagram

```mermaid
sequenceDiagram
    actor O as Organizer
    participant Event as Event Service
    participant Norm as Normalization Service
    participant DB as Database
    participant Audit as Audit Log
    participant Gallery as Gallery (Public)

    %% Pre-flight check
    O->>Event: GET /events/{id}/judging-status
    Event->>DB: SELECT COUNT(assignments) by status
    Event-->>O: { total: 141, completed: 138, pending: 3, recused: 0 }

    %% Trigger normalization
    O->>Norm: POST /events/{id}/normalize
    Norm->>Norm: Check: judging_deadline_at passed OR all completed? (I8)
    
    alt Judging not complete and deadline not passed
        Norm-->>O: 422 "Judging still in progress"
    else Ready to normalize
        Norm->>DB: UPDATE event SET normalization_status='normalizing'
        Norm-->>O: 202 Accepted { job_id }
        
        note over Norm,DB: Background job runs
        
        Norm->>DB: SELECT all scores WHERE event_id = event_id\nAND assignment.status = 'completed'
        Norm->>Norm: GROUP scores by judge_id
        
        loop For each judge j
            Norm->>Norm: Compute μ_j = MEAN(raw_score)
            Norm->>Norm: Compute σ_j = STDDEV(raw_score)
            
            alt σ_j = 0 (judge scored identically)
                Norm->>Norm: Set all normalized_score = 0\nFlag judge in report
            else Normal case
                loop For each score of judge j
                    Norm->>Norm: normalized = (raw - μ_j) / σ_j
                    Norm->>DB: UPDATE score SET normalized_score = normalized
                end
            end
        end
        
        Norm->>DB: Compute weighted_normalized_total per (judge, submission)
        Norm->>DB: Compute final_score per submission\n= AVG(weighted_normalized_total)
        Norm->>DB: Compute rank within each track\n(ORDER BY final_score DESC)
        Norm->>DB: UPDATE event SET normalization_status='completed'
        Norm->>Audit: WRITE normalization.completed
    end

    %% Organizer previews results
    O->>Event: GET /events/{id}/results/preview
    Event->>Event: Check: normalization_status = 'completed'?
    Event->>DB: SELECT submissions with final_score, rank, track
    Event-->>O: Private results view (not public yet)

    %% Publish results
    O->>Event: POST /events/{id}/results/publish
    Event->>Event: Check: normalization_status = 'completed'? (I16)
    Event->>DB: UPDATE event SET\n  status='results_published'\n  results_published_at=now()
    Event->>Audit: WRITE result.published
    Event-->>O: 200 OK

    Gallery-->>PublicUser: Gallery now shows scores, ranks, and winners
```

---

## Normalization Math (Visual)

```
Judge A (Lenient — scores 8, 9, 8.5, 9, 7.5):
  μ_A = 8.4,  σ_A = 0.60
  
  Project X raw=8.0 → normalized = (8.0 - 8.4) / 0.60 = -0.67
  Project Y raw=9.0 → normalized = (9.0 - 8.4) / 0.60 = +1.00

Judge B (Harsh — scores 5, 6, 4.5, 5.5, 7):
  μ_B = 5.6,  σ_B = 0.87
  
  Project X raw=6.0 → normalized = (6.0 - 5.6) / 0.87 = +0.46
  Project Y raw=4.5 → normalized = (4.5 - 5.6) / 0.87 = -1.26

Result: Projects are now comparable across judges.
A lenient judge giving 8 and a harsh judge giving 6 
can be properly weighted together.
```

---

## Results Data Shape

After normalization and publication, each submission has:

```json
{
  "submission_id": "...",
  "title": "Project Alpha",
  "team": "Team Rocket",
  "track": "AI Track",
  "rank": 1,
  "final_score": 0.847,
  "judge_count": 3,
  "criteria_breakdown": [
    { "criterion": "Technical Complexity", "avg_normalized": 1.2, "weight": 0.30 },
    { "criterion": "Innovation",           "avg_normalized": 0.8, "weight": 0.25 },
    { "criterion": "Presentation",         "avg_normalized": 0.6, "weight": 0.20 },
    { "criterion": "Completeness",         "avg_normalized": 0.5, "weight": 0.25 }
  ]
}
```

---

## CSV Export Formats

Organizer can export at any stage:

**Scores export** (`/events/{id}/export/scores.csv`):
```
submission_id,submission_title,judge_id,judge_name,criterion,raw_score,normalized_score,comment
```

**Rankings export** (`/events/{id}/export/rankings.csv`):
```
rank,track,submission_id,title,team_name,final_score,judge_count
```

---

## Error Cases

| Scenario | HTTP Status | Error Code |
|----------|-------------|------------|
| Normalization before judging complete (I8) | 422 | `JUDGING_NOT_COMPLETE` |
| Publish before normalization (I16) | 422 | `NORMALIZATION_NOT_COMPLETE` |
| Normalization already in progress | 409 | `NORMALIZATION_IN_PROGRESS` |
