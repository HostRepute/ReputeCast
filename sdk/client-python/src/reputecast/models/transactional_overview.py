from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_overview_period import TransactionalOverviewPeriod
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_access import TransactionalAccess
  from ..models.transactional_alert import TransactionalAlert
  from ..models.transactional_failover_state import TransactionalFailoverState
  from ..models.transactional_first_run import TransactionalFirstRun
  from ..models.transactional_hour import TransactionalHour
  from ..models.transactional_overview_stats import TransactionalOverviewStats
  from ..models.transactional_route_summary import TransactionalRouteSummary
  from ..models.transactional_sending_key import TransactionalSendingKey





T = TypeVar("T", bound="TransactionalOverview")



@_attrs_define
class TransactionalOverview:
    """ 
        Attributes:
            alerts (list[TransactionalAlert]):
            can (TransactionalAccess):
            endpoint (str):
            first_run (TransactionalFirstRun):
            hour (TransactionalHour):
            incidents (list[TransactionalAlert]):
            include_test (bool):
            keys (list[TransactionalSendingKey]):
            period (TransactionalOverviewPeriod):
            route (TransactionalRouteSummary):
            stats (TransactionalOverviewStats):
            failover (TransactionalFailoverState | Unset):
     """

    alerts: list[TransactionalAlert]
    can: TransactionalAccess
    endpoint: str
    first_run: TransactionalFirstRun
    hour: TransactionalHour
    incidents: list[TransactionalAlert]
    include_test: bool
    keys: list[TransactionalSendingKey]
    period: TransactionalOverviewPeriod
    route: TransactionalRouteSummary
    stats: TransactionalOverviewStats
    failover: TransactionalFailoverState | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_access import TransactionalAccess # noqa: PLC0415
        from ..models.transactional_alert import TransactionalAlert # noqa: PLC0415
        from ..models.transactional_failover_state import TransactionalFailoverState # noqa: PLC0415
        from ..models.transactional_first_run import TransactionalFirstRun # noqa: PLC0415
        from ..models.transactional_hour import TransactionalHour # noqa: PLC0415
        from ..models.transactional_overview_stats import TransactionalOverviewStats # noqa: PLC0415
        from ..models.transactional_route_summary import TransactionalRouteSummary # noqa: PLC0415
        from ..models.transactional_sending_key import TransactionalSendingKey # noqa: PLC0415
        alerts = []
        for alerts_item_data in self.alerts:
            alerts_item = alerts_item_data.to_dict()
            alerts.append(alerts_item)



        can = self.can.to_dict()

        endpoint = self.endpoint

        first_run = self.first_run.to_dict()

        hour = self.hour.to_dict()

        incidents = []
        for incidents_item_data in self.incidents:
            incidents_item = incidents_item_data.to_dict()
            incidents.append(incidents_item)



        include_test = self.include_test

        keys = []
        for keys_item_data in self.keys:
            keys_item = keys_item_data.to_dict()
            keys.append(keys_item)



        period = self.period.value

        route = self.route.to_dict()

        stats = self.stats.to_dict()

        failover: dict[str, Any] | Unset = UNSET
        if not isinstance(self.failover, Unset):
            failover = self.failover.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alerts": alerts,
            "can": can,
            "endpoint": endpoint,
            "firstRun": first_run,
            "hour": hour,
            "incidents": incidents,
            "includeTest": include_test,
            "keys": keys,
            "period": period,
            "route": route,
            "stats": stats,
        })
        if failover is not UNSET:
            field_dict["failover"] = failover

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_access import TransactionalAccess # noqa: PLC0415
        from ..models.transactional_alert import TransactionalAlert # noqa: PLC0415
        from ..models.transactional_failover_state import TransactionalFailoverState # noqa: PLC0415
        from ..models.transactional_first_run import TransactionalFirstRun # noqa: PLC0415
        from ..models.transactional_hour import TransactionalHour # noqa: PLC0415
        from ..models.transactional_overview_stats import TransactionalOverviewStats # noqa: PLC0415
        from ..models.transactional_route_summary import TransactionalRouteSummary # noqa: PLC0415
        from ..models.transactional_sending_key import TransactionalSendingKey # noqa: PLC0415
        d = dict(src_dict)
        alerts = []
        _alerts = d.pop("alerts")
        for alerts_item_data in (_alerts):
            alerts_item = TransactionalAlert.from_dict(alerts_item_data)



            alerts.append(alerts_item)


        can = TransactionalAccess.from_dict(d.pop("can"))




        endpoint = d.pop("endpoint")

        first_run = TransactionalFirstRun.from_dict(d.pop("firstRun"))




        hour = TransactionalHour.from_dict(d.pop("hour"))




        incidents = []
        _incidents = d.pop("incidents")
        for incidents_item_data in (_incidents):
            incidents_item = TransactionalAlert.from_dict(incidents_item_data)



            incidents.append(incidents_item)


        include_test = d.pop("includeTest")

        keys = []
        _keys = d.pop("keys")
        for keys_item_data in (_keys):
            keys_item = TransactionalSendingKey.from_dict(keys_item_data)



            keys.append(keys_item)


        period = TransactionalOverviewPeriod(d.pop("period"))




        route = TransactionalRouteSummary.from_dict(d.pop("route"))




        stats = TransactionalOverviewStats.from_dict(d.pop("stats"))




        _failover = d.pop("failover", UNSET)
        failover: TransactionalFailoverState | Unset
        if isinstance(_failover,  Unset):
            failover = UNSET
        else:
            failover = TransactionalFailoverState.from_dict(_failover)




        transactional_overview = cls(
            alerts=alerts,
            can=can,
            endpoint=endpoint,
            first_run=first_run,
            hour=hour,
            incidents=incidents,
            include_test=include_test,
            keys=keys,
            period=period,
            route=route,
            stats=stats,
            failover=failover,
        )

        return transactional_overview

