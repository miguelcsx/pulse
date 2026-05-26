# Pulse 2.0: Human Advice Network

Pulse is the human layer after AI.

AI gives answers. Pulse finds the human who has lived your question.

## Product Thesis

People increasingly ask LLMs for help, but many important moments still need
judgment, lived experience, emotional calibration, accountability, or trust from
another person. Pulse turns asks, proof moments, reactions, and help outcomes
into an explainable affinity graph for mentor and peer connection.

## Core Loop

1. A user writes an ask: a question, dilemma, or situation where they need human
   perspective.
2. Pulse triages the ask into topic, urgency, and desired help type.
3. Pulse generates human bridges: mentor, peer, and adjacent perspective.
4. The user opens a bridge, joins a live help room, or saves the person for
   later.
5. Help signals improve future matching.

## Product Surfaces

- Today: the home connection console. No infinite feed as the default product.
- Human bridges: compact, explainable matches with fast actions.
- Live help rooms: temporary rooms around intent, not generic hashtags.
- Trust profile: topics, lived experience, availability, helped count, and
  response quality.
- Proof moments: content remains available as evidence and context, not the
  main consumption loop.

## Production Direction

- Postgres and pgvector are the primary production matching path.
- Postgres and pgvector are the only production matching path.
- Ollama remains the free/local embedding provider behind the existing embedder
  interface.
- The matching algorithm ranks useful humans for a need, not attractive posts.
