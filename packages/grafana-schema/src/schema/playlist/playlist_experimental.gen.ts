// This file is autogenerated. DO NOT EDIT.
//
// Generated by pkg/framework/coremodel/gen.go
//
// Derived from the Thema lineage declared in pkg/coremodel/playlist/coremodel.cue
//
// Run `make gen-cue` from repository root to regenerate.


// This model is a WIP and not yet canonical. Consequently, its members are
// not exported to exclude it from grafana-schema's public API surface.

interface Playlist {
  interval: string;
  items?: {
    type: ('dashboard_by_uid' | 'dashboard_by_tag' | 'dashboard_by_id');
    value: string;
  }[];
  name: string;
  uid: string;
}

const defaultPlaylist: Partial<Playlist> = {
  interval: '5m',
  items: [],
};