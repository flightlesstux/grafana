import { extend } from 'lodash';
import React, { PureComponent } from 'react';

import { AppEvents, PluginMeta, DataSourceApi } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { appEvents } from 'app/core/core';
import DashboardsTable from 'app/features/datasources/DashboardsTable';
import { PluginDashboard } from 'app/types';

interface Props {
  plugin: PluginMeta;
  datasource?: DataSourceApi;
}

interface State {
  dashboards: PluginDashboard[];
  loading: boolean;
}

export class PluginDashboards extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      loading: true,
      dashboards: [],
    };
  }

  async componentDidMount() {
    const pluginId = this.props.plugin.id;
    getBackendSrv()
      .get(`/api/plugins/${pluginId}/dashboards`)
      .then((dashboards: any) => {
        this.setState({ dashboards, loading: false });
      });
  }

  importAll = () => {
    this.importNext(0);
  };

  private importNext = (index: number) => {
    const { dashboards } = this.state;
    return this.import(dashboards[index], true).then(() => {
      if (index + 1 < dashboards.length) {
        return new Promise<void>((resolve) => {
          setTimeout(() => {
            this.importNext(index + 1).then(() => {
              resolve();
            });
          }, 500);
        });
      } else {
        return Promise.resolve();
      }
    });
  };

  import = (dash: PluginDashboard, overwrite: boolean) => {
    const { plugin, datasource } = this.props;

    const installCmd: any = {
      pluginId: plugin.id,
      path: dash.path,
      overwrite: overwrite,
      inputs: [],
    };

    if (datasource) {
      installCmd.inputs.push({
        name: '*',
        type: 'datasource',
        pluginId: datasource.meta.id,
        value: datasource.name,
      });
    }

    return getBackendSrv()
      .post(`/api/dashboards/import`, installCmd)
      .then((res: PluginDashboard) => {
        appEvents.emit(AppEvents.alertSuccess, ['Dashboard Imported', dash.title]);
        extend(dash, res);
        this.setState({ dashboards: [...this.state.dashboards] });
      });
  };

  remove = (dash: PluginDashboard) => {
    getBackendSrv()
      .delete('/api/dashboards/uid/' + dash.uid)
      .then(() => {
        dash.imported = false;
        this.setState({ dashboards: [...this.state.dashboards] });
      });
  };

  render() {
    const { loading, dashboards } = this.state;
    if (loading) {
      return <div>loading...</div>;
    }
    if (!dashboards || !dashboards.length) {
      return <div>No dashboards are included with this plugin</div>;
    }

    return (
      <div className="gf-form-group">
        <DashboardsTable dashboards={dashboards} onImport={this.import} onRemove={this.remove} />
      </div>
    );
  }
}
