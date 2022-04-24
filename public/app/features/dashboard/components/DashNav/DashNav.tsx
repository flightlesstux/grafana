import React, { FC, ReactNode } from 'react';
import { connect, ConnectedProps } from 'react-redux';

import { NavModel, textUtil } from '@grafana/data';
import { locationService } from '@grafana/runtime';
import { ButtonGroup, ModalsController, ToolbarButton, PageToolbar, useForceUpdate, Button } from '@grafana/ui';
import appEvents from 'app/core/app_events';
import { ToggleMegaMenu } from 'app/core/components/NavBar/Next/MegaMenu';
import config from 'app/core/config';
import { toggleKioskMode } from 'app/core/navigation/kiosk';
import { DashboardCommentsModal } from 'app/features/dashboard/components/DashboardComments/DashboardCommentsModal';
import { ShareModal } from 'app/features/dashboard/components/ShareModal';
import { playlistSrv } from 'app/features/playlist/PlaylistSrv';
import { updateTimeZoneForSession } from 'app/features/profile/state/reducers';
import { KioskMode } from 'app/types';

import { getDashboardSrv } from '../../services/DashboardSrv';
import { DashboardModel } from '../../state';
import { SaveDashboardDrawer } from '../SaveDashboard/SaveDashboardDrawer';

import { DashNavButton } from './DashNavButton';
import { DashNavTimeControls } from './DashNavTimeControls';

const mapDispatchToProps = {
  updateTimeZoneForSession,
};

const connector = connect(null, mapDispatchToProps);

export interface OwnProps {
  dashboard: DashboardModel;
  isFullscreen: boolean;
  kioskMode: KioskMode;
  hideTimePicker: boolean;
  editview?: string;
  navModel: NavModel;
  folderTitle?: string;
  title: string;
  onAddPanel: () => void;
}

interface DashNavButtonModel {
  show: (props: Props) => boolean;
  component: FC<Partial<Props>>;
  index?: number | 'end';
}

const customLeftActions: DashNavButtonModel[] = [];
const customRightActions: DashNavButtonModel[] = [];

export function addCustomLeftAction(content: DashNavButtonModel) {
  customLeftActions.push(content);
}

export function addCustomRightAction(content: DashNavButtonModel) {
  customRightActions.push(content);
}

type Props = OwnProps & ConnectedProps<typeof connector>;

export const DashNav = React.memo<Props>((props) => {
  const forceUpdate = useForceUpdate();

  const onStarDashboard = () => {
    const dashboardSrv = getDashboardSrv();
    const { dashboard } = props;

    dashboardSrv.starDashboard(dashboard.id, dashboard.meta.isStarred).then((newState: any) => {
      dashboard.meta.isStarred = newState;
      forceUpdate();
    });
  };

  const onClose = () => {
    locationService.partial({ viewPanel: null });
  };

  const onToggleTVMode = () => {
    toggleKioskMode();
  };

  const onOpenSettings = () => {
    locationService.partial({ editview: 'settings' });
  };

  const onCloseSettings = () => {
    locationService.partial({ editview: null, editIndex: null });
  };

  const onPlaylistPrev = () => {
    playlistSrv.prev();
  };

  const onPlaylistNext = () => {
    playlistSrv.next();
  };

  const onPlaylistStop = () => {
    playlistSrv.stop();
    forceUpdate();
  };

  const addCustomContent = (actions: DashNavButtonModel[], buttons: ReactNode[]) => {
    actions.map((action, index) => {
      const Component = action.component;
      const element = <Component {...props} key={`button-custom-${index}`} />;
      typeof action.index === 'number' ? buttons.splice(action.index, 0, element) : buttons.push(element);
    });
  };

  const isPlaylistRunning = () => {
    return playlistSrv.isPlaying;
  };

  const renderLeftActionsButton = () => {
    const { dashboard, kioskMode } = props;
    const { canStar, canShare, isStarred } = dashboard.meta;
    const buttons: ReactNode[] = [];

    if (kioskMode !== KioskMode.Off || isPlaylistRunning()) {
      return [];
    }

    if (canStar) {
      let desc = isStarred ? 'Unmark as favorite' : 'Mark as favorite';
      buttons.push(
        <DashNavButton
          tooltip={desc}
          icon={isStarred ? 'favorite' : 'star'}
          iconType={isStarred ? 'mono' : 'default'}
          iconSize="lg"
          onClick={onStarDashboard}
          key="button-star"
        />
      );
    }

    if (canShare) {
      let desc = 'Share dashboard or panel';
      buttons.push(
        <ModalsController key="button-share">
          {({ showModal, hideModal }) => (
            <DashNavButton
              tooltip={desc}
              icon="share-alt"
              iconSize="lg"
              onClick={() => {
                showModal(ShareModal, {
                  dashboard,
                  onDismiss: hideModal,
                });
              }}
            />
          )}
        </ModalsController>
      );
    }

    if (dashboard.uid && config.featureToggles.dashboardComments) {
      buttons.push(
        <ModalsController key="button-dashboard-comments">
          {({ showModal, hideModal }) => (
            <DashNavButton
              tooltip="Show dashboard comments"
              icon="comment-alt-message"
              iconSize="lg"
              onClick={() => {
                showModal(DashboardCommentsModal, {
                  dashboard,
                  onDismiss: hideModal,
                });
              }}
            />
          )}
        </ModalsController>
      );
    }

    addCustomContent(customLeftActions, buttons);
    return buttons;
  };

  const renderPlaylistControls = () => {
    return (
      <ButtonGroup key="playlist-buttons">
        <ToolbarButton tooltip="Go to previous dashboard" icon="backward" onClick={onPlaylistPrev} narrow />
        <ToolbarButton onClick={onPlaylistStop}>Stop playlist</ToolbarButton>
        <ToolbarButton tooltip="Go to next dashboard" icon="forward" onClick={onPlaylistNext} narrow />
      </ButtonGroup>
    );
  };

  const renderTimeControls = () => {
    const { dashboard, updateTimeZoneForSession, hideTimePicker } = props;

    if (hideTimePicker) {
      return null;
    }

    return (
      <DashNavTimeControls dashboard={dashboard} onChangeTimeZone={updateTimeZoneForSession} key="time-controls" />
    );
  };

  const renderRightActionsButton = () => {
    const { dashboard, onAddPanel, isFullscreen, kioskMode, editview } = props;
    const { canSave, canEdit, showSettings } = dashboard.meta;
    const { snapshot } = dashboard;
    const snapshotUrl = snapshot && snapshot.originalUrl;
    const buttons: ReactNode[] = [];
    const tvButton = (
      <ToolbarButton tooltip="Cycle view mode" icon="monitor" onClick={onToggleTVMode} key="tv-button" />
    );

    if (isPlaylistRunning()) {
      return [renderPlaylistControls(), renderTimeControls()];
    }

    if (kioskMode === KioskMode.TV) {
      return [renderTimeControls(), tvButton];
    }

    if (canEdit && !isFullscreen && !editview) {
      buttons.push(<ToolbarButton tooltip="Add panel" icon="panel-add" onClick={onAddPanel} key="button-panel-add" />);
    }

    if (canSave && !isFullscreen && !editview) {
      buttons.push(
        <ModalsController key="button-save">
          {({ showModal, hideModal }) => (
            <ToolbarButton
              tooltip="Save dashboard"
              icon="save"
              onClick={() => {
                showModal(SaveDashboardDrawer, {
                  dashboard,
                  onDismiss: hideModal,
                });
              }}
            />
          )}
        </ModalsController>
      );
    }

    if (snapshotUrl) {
      buttons.push(
        <ToolbarButton
          tooltip="Open original dashboard"
          onClick={() => gotoSnapshotOrigin(snapshotUrl)}
          icon="link"
          key="button-snapshot"
        />
      );
    }

    if (!editview) {
      if (showSettings) {
        buttons.push(
          <ToolbarButton
            tooltip="Dashboard settings"
            icon="cog"
            onClick={onOpenSettings}
            key="button-settings"
            variant={editview ? 'active' : 'default'}
          />
        );
      }

      addCustomContent(customRightActions, buttons);
      buttons.push(renderTimeControls());
      buttons.push(tvButton);
    } else {
      buttons.push(
        <Button icon="arrow-left" onClick={onCloseSettings} key="button-back" variant="secondary" size="sm">
          Exit settings
        </Button>
      );
      buttons.push(
        <ModalsController key="button-save">
          {({ showModal, hideModal }) => (
            <Button
              size="sm"
              onClick={() => {
                showModal(SaveDashboardDrawer, {
                  dashboard,
                  onDismiss: hideModal,
                });
              }}
            >
              Save dashboard
            </Button>
          )}
        </ModalsController>
      );
    }

    return buttons;
  };

  const gotoSnapshotOrigin = (snapshotUrl: string) => {
    window.location.href = textUtil.sanitizeUrl(snapshotUrl);
  };

  const { isFullscreen } = props;
  // this ensures the component rerenders when the location changes
  // const location = useLocation();
  // const titleHref = locationUtil.getUrlForPartial(location, { search: 'open' });
  // const parentHref = locationUtil.getUrlForPartial(location, { search: 'open', folder: 'current' });
  const onGoBack = isFullscreen ? onClose : undefined;

  return (
    <PageToolbar
      navModel={props.navModel.node}
      onOpenMenu={() => appEvents.publish(new ToggleMegaMenu())}
      onGoBack={onGoBack}
      leftItems={renderLeftActionsButton()}
    >
      {renderRightActionsButton()}
    </PageToolbar>
  );
});

DashNav.displayName = 'DashNav';

export default connector(DashNav);
