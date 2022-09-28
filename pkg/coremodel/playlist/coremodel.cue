package playlist

import (
	"github.com/grafana/thema"
)

thema.#Lineage
name: "playlist"
seqs: [
	{
		schemas: [
			{//0.0
				// Unique playlist identifier. Generated on creation, either by the
				// creator of the playlist of by the application.
				uid: string

				// Name of the playlist.
				name: string

				// Interval sets the time between switching views in a playlist.
				// The format is read by https://github.com/grafana/grafana/blob/v9.2.x/packages/grafana-data/src/datetime/rangeutil.ts#L332
				interval: string | *"5m"

				// The ordered list of items that the playlist will iterate over.
				items?: [...#PlaylistItem]

				///////////////////////////////////////
				// Definitions (referenced above) are declared below

				#PlaylistItem: {
					// Type of the item. `dashboard_by_id` is deprecated
					type: "dashboard_by_uid" | "dashboard_by_tag" | "dashboard_by_id"

					// Value depends on type and describes the playlist item.
					//
					//  - dashboard_by_id: The value is an internal numerical identifier set by Grafana. This
					//  is not portable as the numerical identifier is non-deterministic between different instances.
					//  Will be replaced by dashboard_by_uid in the future.
					//  - dashboard_by_uid: A unique id for the dashboard.
					//  - dashboard_by_tag: The value is a tag which is set on any number of dashboards. All
					//  dashboards behind the tag will be added to the playlist.
					value: string
				}
			}
		]
	}
]
