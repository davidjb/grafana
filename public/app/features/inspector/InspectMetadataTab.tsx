import React from 'react';
import { Trans } from 'react-i18next';

import { DataSourceApi, PanelData } from '@grafana/data';

interface InspectMetadataTabProps {
  data: PanelData;
  metadataDatasource?: DataSourceApi;
}
export const InspectMetadataTab: React.FC<InspectMetadataTabProps> = ({ data, metadataDatasource }) => {
  if (!metadataDatasource || !metadataDatasource.components?.MetadataInspector) {
    return <Trans i18nKey="dashboard.inspect-meta.no-inspector">No Metadata Inspector</Trans>;
  }
  return <metadataDatasource.components.MetadataInspector datasource={metadataDatasource} data={data.series} />;
};
