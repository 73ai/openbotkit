package main

import "log/slog"

func Seed(st *Store) error {
	author, err := st.UpsertUser("seed-author", "registry@openbotkit.dev", "OpenBotKit Team", "")
	if err != nil {
		return err
	}

	seeds := []UseCase{
		{
			ID:                newID(),
			Title:             "Customer Support Ticket Triage",
			Description:       "Automatically classify and route incoming support tickets by urgency, topic, and required expertise. Reduces first-response time and ensures tickets reach the right team immediately.",
			Domain:            "Customer Support",
			IndustryTags:      "SaaS,E-commerce,Fintech",
			RiskLevel:         "low",
			ROIPotential:      "high",
			Status:            "published",
			ImplStatus:        "production",
			Visibility:        "public",
			SafetyPII:         true,
			SafetyAutonomous:  false,
			SafetyBlastRadius: "Low - misrouted tickets get manually corrected",
			SafetyOversight:   "Human reviews all urgent/escalated classifications",
			AuthorID:          author.ID,
		},
		{
			ID:                newID(),
			Title:             "Invoice Data Extraction",
			Description:       "Extract structured data (vendor, amount, line items, dates) from PDF invoices and feed into accounting systems. Eliminates manual data entry for accounts payable.",
			Domain:            "Finance",
			IndustryTags:      "Accounting,Enterprise,SMB",
			RiskLevel:         "medium",
			ROIPotential:      "high",
			Status:            "published",
			ImplStatus:        "production",
			Visibility:        "public",
			SafetyPII:         true,
			SafetyAutonomous:  false,
			SafetyBlastRadius: "Medium - incorrect extraction could cause payment errors",
			SafetyOversight:   "Human approves all extractions above $1000 threshold",
			AuthorID:          author.ID,
		},
		{
			ID:                newID(),
			Title:             "Meeting Notes Summarization",
			Description:       "Generate structured meeting summaries with action items, decisions, and key discussion points from transcripts. Distribute to attendees automatically after each meeting.",
			Domain:            "Productivity",
			IndustryTags:      "Enterprise,Startup,Remote Work",
			RiskLevel:         "low",
			ROIPotential:      "medium",
			Status:            "published",
			ImplStatus:        "production",
			Visibility:        "public",
			SafetyPII:         false,
			SafetyAutonomous:  false,
			SafetyBlastRadius: "Low - summaries are supplementary, not authoritative",
			SafetyOversight:   "Meeting organizer reviews before distribution",
			AuthorID:          author.ID,
		},
		{
			ID:                newID(),
			Title:             "Content Moderation for User-Generated Content",
			Description:       "Screen user-submitted text, images, and comments for policy violations, hate speech, and spam. Flag violations for human review with confidence scores.",
			Domain:            "Trust & Safety",
			IndustryTags:      "Social Media,Marketplace,Community",
			RiskLevel:         "high",
			ROIPotential:      "high",
			Status:            "published",
			ImplStatus:        "evaluating",
			Visibility:        "public",
			SafetyPII:         true,
			SafetyAutonomous:  true,
			SafetyBlastRadius: "High - false positives silence legitimate users, false negatives expose harmful content",
			SafetyOversight:   "All automated removals have 24h appeal window; borderline cases always go to human review",
			AuthorID:          author.ID,
		},
		{
			ID:                newID(),
			Title:             "Sales Email Personalization",
			Description:       "Generate personalized outreach emails based on prospect company data, recent news, and product fit analysis. Sales rep reviews and sends.",
			Domain:            "Sales",
			IndustryTags:      "SaaS,B2B,Enterprise",
			RiskLevel:         "low",
			ROIPotential:      "medium",
			Status:            "published",
			ImplStatus:        "production",
			Visibility:        "public",
			SafetyPII:         false,
			SafetyAutonomous:  false,
			SafetyBlastRadius: "Low - human always reviews before sending",
			SafetyOversight:   "Sales rep manually reviews and sends every email",
			AuthorID:          author.ID,
		},
		{
			ID:                newID(),
			Title:             "Medical Record Coding (ICD-10)",
			Description:       "Suggest ICD-10 diagnosis codes from clinical notes to assist medical coders. Reduces coding time and improves accuracy for insurance billing.",
			Domain:            "Healthcare",
			IndustryTags:      "Healthcare,Insurance,Hospital",
			RiskLevel:         "high",
			ROIPotential:      "high",
			Status:            "published",
			ImplStatus:        "evaluating",
			Visibility:        "public",
			SafetyPII:         true,
			SafetyAutonomous:  false,
			SafetyBlastRadius: "High - incorrect codes affect patient billing and insurance claims",
			SafetyOversight:   "Certified medical coder reviews and approves every code suggestion",
			AuthorID:          author.ID,
		},
		{
			ID:                newID(),
			Title:             "Automated Code Review Comments",
			Description:       "Analyze pull requests for common issues: security vulnerabilities, style violations, performance problems, and missing tests. Post review comments automatically.",
			Domain:            "Engineering",
			IndustryTags:      "SaaS,DevTools,Enterprise",
			RiskLevel:         "low",
			ROIPotential:      "medium",
			Status:            "published",
			ImplStatus:        "production",
			Visibility:        "public",
			SafetyPII:         false,
			SafetyAutonomous:  false,
			SafetyBlastRadius: "Low - comments are suggestions, not blocking",
			SafetyOversight:   "Developers choose which suggestions to accept",
			AuthorID:          author.ID,
		},
		{
			ID:                newID(),
			Title:             "Lease Agreement Analysis",
			Description:       "Extract key terms, obligations, renewal dates, and risk clauses from commercial lease agreements. Generate structured summaries for legal and real estate teams.",
			Domain:            "Legal",
			IndustryTags:      "Real Estate,Legal,Enterprise",
			RiskLevel:         "medium",
			ROIPotential:      "high",
			Status:            "published",
			ImplStatus:        "evaluating",
			Visibility:        "public",
			SafetyPII:         true,
			SafetyAutonomous:  false,
			SafetyBlastRadius: "Medium - missed clauses could have financial consequences",
			SafetyOversight:   "Legal counsel reviews all AI-generated summaries before action",
			AuthorID:          author.ID,
		},
	}

	for i := range seeds {
		seeds[i].Slug = slugify(seeds[i].Title) + "-" + seeds[i].ID[:8]
		if err := st.CreateUseCase(&seeds[i]); err != nil {
			slog.Warn("seed: skip existing", "title", seeds[i].Title, "error", err)
			continue
		}
		slog.Info("seed: created", "title", seeds[i].Title)
	}

	return nil
}
