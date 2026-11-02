# ==========================================
# NimmaSaarthi Documentation Bootstrap Script
# ==========================================

Write-Host "Creating NimmaSaarthi documentation..."

$base = Get-Location

New-Item -ItemType Directory -Force -Path "docs/vision" | Out-Null
New-Item -ItemType Directory -Force -Path "docs/architecture" | Out-Null

# =====================================================
# EXECUTIVE SUMMARY
# =====================================================

@'
# NimmaSaarthi

## Metro-Aware Mobility Orchestration Platform

### Tagline

Match Journeys, Not Destinations.

---

## Vision

NimmaSaarthi is a mobility orchestration platform that optimizes the transition between metro travel and final destination travel.

Instead of forcing commuters to search for transportation after leaving a station, NimmaSaarthi predicts metro arrivals, forms efficient rider pools before arrival, and coordinates vehicles in advance.

---

## Core Innovation

Traditional ride sharing:

Request Ride
→ Match Driver
→ Wait
→ Travel

NimmaSaarthi:

Board Train
→ Pool Me
→ Corridor Matching
→ Pre-position Vehicle
→ Exit Station
→ Board Ride

---

## Mission

NimmaSaarthi coordinates passengers, vehicles and transit systems to create seamless last-mile transportation.
'@ | Set-Content "docs/vision/executive-summary.md"

# =====================================================
# PRODUCT VISION
# =====================================================

@'
# Product Vision

## NimmaSaarthi

NimmaSaarthi is a Metro-Aware Mobility Orchestration Platform.

---

## Guiding Principle

Match Journeys, Not Destinations.

---

## Core Features

### Pool Me

Request pooling before arrival.

### Journey Matching

Corridor-based matching instead of identical destinations.

### Vehicle Assignment

Dynamic allocation of autos, cabs and shuttles.

### Fair Fare Engine

Transparent pricing using contribution-based allocation.

### SafHER

Safety-first travel experiences.

### WalkFlex

Discounts for riders willing to walk small distances.

---

## Success Metrics

- Pool Formation Rate
- Average Savings
- Wait Time
- Driver Revenue
- Vehicle Trips Saved
'@ | Set-Content "docs/vision/product-vision.md"

# =====================================================
# PROBLEM STATEMENT
# =====================================================

@'
# Problem Statement

## The Last Mile Problem

Mass transit efficiently moves people across cities.

The challenge begins when commuters leave the station.

---

## Current Flow

Metro Arrival
↓
Exit Station
↓
Search For Ride
↓
Wait
↓
Travel

---

## Problems

### Riders

- High costs
- Long waits
- Safety concerns
- Unpredictable experience

### Drivers

- Idle time
- Poor utilization

### Cities

- Congestion
- Low occupancy
- Fragmented mobility

---

## Opportunity

Metro commuter flows are predictable.

NimmaSaarthi uses this predictability to coordinate transportation before commuters arrive at station exits.
'@ | Set-Content "docs/vision/problem-statement.md"

# =====================================================
# BUSINESS MODEL
# =====================================================

@'
# Business Model

## Revenue Streams

### Platform Commission

Percentage of pooled ride revenue.

### Subscription Plans

Monthly commuter passes.

### Sponsored Bays

Branding opportunities at pickup locations.

### Enterprise Mobility

Corporate mobility partnerships.

---

## Cost Structure

- Driver incentives
- Platform infrastructure
- Safety operations
- Customer support
'@ | Set-Content "docs/vision/business-model.md"

# =====================================================
# ROADMAP
# =====================================================

@'
# Roadmap

## Phase 1

Simulation MVP

## Phase 2

Campus Pilot

## Phase 3

Metro Pilot

## Phase 4

Multi-Station Expansion

## Phase 5

National Mobility Platform
'@ | Set-Content "docs/vision/roadmap.md"

# =====================================================
# SYSTEM OVERVIEW
# =====================================================

@'
# System Overview

## Architecture Philosophy

Simulation First.
Production Ready.

---

## Experience Layer

- Rider App
- Driver App
- Ops Dashboard
- Admin Dashboard

---

## Intelligence Layer

### ETA Engine

Predicts arrivals.

### Matching Engine

Forms rider groups.

### Routing Engine

Computes drop sequence.

### Fare Engine

Calculates fair pricing.

### Safety Engine

Manages SafHER policies.

---

## Infrastructure Layer

- Frontend
- Go Backend
- PostgreSQL
- Redis
- WebSockets

---

## Simulation Layer

- Train Simulation
- Rider Simulation
- Driver Simulation
- Analytics Engine
'@ | Set-Content "docs/architecture/system-overview.md"

# =====================================================
# FRONTEND ARCHITECTURE
# =====================================================

@'
# Frontend Architecture

## Stack

- HTML
- CSS
- JavaScript
- Canvas
- SVG
- Web Workers

## Core Flows

### Rider

Login
→ Pool Request
→ Matching
→ Bay Assignment
→ Live Trip
→ Payment
→ Rating

### Driver

Login
→ Assignment
→ Pickup
→ Navigation
→ Earnings

### Admin

Dashboard
→ Users
→ Trips
→ Analytics
→ Reports
'@ | Set-Content "docs/architecture/frontend-architecture.md"

# =====================================================
# BACKEND ARCHITECTURE
# =====================================================

@'
# Backend Architecture

## Language

Go

## Database

PostgreSQL

## Cache

Redis

## Core Services

- Authentication
- Matching
- Pooling
- Routing
- Fare Engine
- Payments
- Wallet
- Trips
- Analytics
- Notifications

## Communication

REST API

WebSocket Realtime Updates
'@ | Set-Content "docs/architecture/backend-architecture.md"

Write-Host "Documentation created successfully."

# ==========================================
# GIT
# ==========================================

git add .

git commit -m "docs: bootstrap NimmaSaarthi documentation"

git push origin main

Write-Host ""
Write-Host "NimmaSaarthi documentation pushed successfully." -ForegroundColor Green