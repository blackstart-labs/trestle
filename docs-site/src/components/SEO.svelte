<script lang="ts">
  import { getSectionSEO } from "../content";

  interface Props {
    sectionId: string;
  }

  let { sectionId }: Props = $props();

  let seo = $derived(getSectionSEO(sectionId));

  let jsonLd = $derived(
    sectionId === "home"
      ? JSON.stringify({
          "@context": "https://schema.org",
          "@graph": [
            {
              "@type": "WebSite",
              "@id": "https://tresstle.vercel.app/#website",
              "url": "https://tresstle.vercel.app/",
              "name": "Trestle Documentation",
              "description": "Self-hosted workspace for projects, tasks, workflows, and knowledge.",
              "inLanguage": "en-US"
            },
            {
              "@type": "SoftwareApplication",
              "@id": "https://tresstle.vercel.app/#software",
              "name": "Trestle",
              "applicationCategory": "BusinessApplication, ProjectManagementApplication",
              "operatingSystem": "Linux, macOS, Windows, Docker",
              "url": "https://tresstle.vercel.app",
              "description": "Self-hosted operational workspace for canonical tasks, Kanban boards, milestone roadmaps, and live time tracking.",
              "offers": {
                "@type": "Offer",
                "price": "0",
                "priceCurrency": "USD"
              },
              "featureList": [
                "Interactive Kanban Boards",
                "Milestone Roadmaps & Gantt Timelines",
                "Live Persistent Task Time Tracking",
                "Isolated Multi-Tenant Workspaces",
                "Telegram Bot Topic-Aware Notifications",
                "Slack and Discord Webhook Alerts",
                "Full JSON Project & Workspace Data Export",
                "Zero Third-Party Dependency Single Binary"
              ]
            }
          ]
        })
      : JSON.stringify({
          "@context": "https://schema.org",
          "@graph": [
            {
              "@type": "TechArticle",
              "@id": `https://tresstle.vercel.app/#${sectionId}`,
              "headline": seo.title,
              "description": seo.description,
              "url": seo.canonical,
              "inLanguage": "en-US",
              "articleSection": seo.category,
              "isPartOf": {
                "@type": "WebSite",
                "name": "Trestle Documentation",
                "url": "https://tresstle.vercel.app"
              },
              "publisher": {
                "@type": "Organization",
                "name": "Trestle",
                "url": "https://tresstle.vercel.app"
              }
            },
            {
              "@type": "BreadcrumbList",
              "itemListElement": [
                {
                  "@type": "ListItem",
                  "position": 1,
                  "name": "Home",
                  "item": "https://tresstle.vercel.app/"
                },
                {
                  "@type": "ListItem",
                  "position": 2,
                  "name": seo.category,
                  "item": `https://tresstle.vercel.app/#${sectionId}`
                },
                {
                  "@type": "ListItem",
                  "position": 3,
                  "name": seo.title,
                  "item": seo.canonical
                }
              ]
            }
          ]
        })
  );
</script>

<svelte:head>
  <title>{seo.title}</title>
  <meta name="description" content={seo.description} />
  <meta name="keywords" content={seo.keywords} />
  <meta name="robots" content="index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1" />
  <link rel="canonical" href={seo.canonical} />

  <!-- Open Graph / Facebook -->
  <meta property="og:type" content={seo.ogType} />
  <meta property="og:url" content={seo.canonical} />
  <meta property="og:title" content={seo.title} />
  <meta property="og:description" content={seo.description} />
  <meta property="og:site_name" content="Trestle Documentation" />
  <meta property="og:image" content="https://tresstle.vercel.app/assets/docs.webp" />
  <meta property="og:image:alt" content="Trestle Documentation Preview" />
  <meta property="og:locale" content="en_US" />

  <!-- Twitter -->
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:url" content={seo.canonical} />
  <meta name="twitter:title" content={seo.title} />
  <meta name="twitter:description" content={seo.description} />
  <meta name="twitter:image" content="https://tresstle.vercel.app/assets/docs.webp" />

  <!-- Schema.org JSON-LD Structured Data -->
  {@html `<script type="application/ld+json">${jsonLd}</script>`}
</svelte:head>

