from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.health_tier_card import HealthTierCard
  from ..models.health_trend_week import HealthTrendWeek
  from ..models.list_health_insight import ListHealthInsight
  from ..models.list_health_row import ListHealthRow
  from ..models.sunset_policy import SunsetPolicy
  from ..models.sunset_this_week import SunsetThisWeek





T = TypeVar("T", bound="ListHealth")



@_attrs_define
class ListHealth:
    """ 
        Attributes:
            policy (SunsetPolicy):
            policy_week (int): Index of the trend column when the policy started, -1 if none
            rows (list[ListHealthRow]):
            this_week (SunsetThisWeek):
            tiers (list[HealthTierCard]):
            total (int):
            trend (list[HealthTrendWeek]):
            computed_at (datetime.datetime | Unset):
            insight (ListHealthInsight | Unset):
            reengage_flow_id (str | Unset):
            win_back_rate (float | Unset):
     """

    policy: SunsetPolicy
    policy_week: int
    rows: list[ListHealthRow]
    this_week: SunsetThisWeek
    tiers: list[HealthTierCard]
    total: int
    trend: list[HealthTrendWeek]
    computed_at: datetime.datetime | Unset = UNSET
    insight: ListHealthInsight | Unset = UNSET
    reengage_flow_id: str | Unset = UNSET
    win_back_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.health_tier_card import HealthTierCard # noqa: PLC0415
        from ..models.health_trend_week import HealthTrendWeek # noqa: PLC0415
        from ..models.list_health_insight import ListHealthInsight # noqa: PLC0415
        from ..models.list_health_row import ListHealthRow # noqa: PLC0415
        from ..models.sunset_policy import SunsetPolicy # noqa: PLC0415
        from ..models.sunset_this_week import SunsetThisWeek # noqa: PLC0415
        policy = self.policy.to_dict()

        policy_week = self.policy_week

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        this_week = self.this_week.to_dict()

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        total = self.total

        trend = []
        for trend_item_data in self.trend:
            trend_item = trend_item_data.to_dict()
            trend.append(trend_item)



        computed_at: str | Unset = UNSET
        if not isinstance(self.computed_at, Unset):
            computed_at = self.computed_at.isoformat()

        insight: dict[str, Any] | Unset = UNSET
        if not isinstance(self.insight, Unset):
            insight = self.insight.to_dict()

        reengage_flow_id = self.reengage_flow_id

        win_back_rate = self.win_back_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "policy": policy,
            "policyWeek": policy_week,
            "rows": rows,
            "thisWeek": this_week,
            "tiers": tiers,
            "total": total,
            "trend": trend,
        })
        if computed_at is not UNSET:
            field_dict["computedAt"] = computed_at
        if insight is not UNSET:
            field_dict["insight"] = insight
        if reengage_flow_id is not UNSET:
            field_dict["reengageFlowId"] = reengage_flow_id
        if win_back_rate is not UNSET:
            field_dict["winBackRate"] = win_back_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.health_tier_card import HealthTierCard # noqa: PLC0415
        from ..models.health_trend_week import HealthTrendWeek # noqa: PLC0415
        from ..models.list_health_insight import ListHealthInsight # noqa: PLC0415
        from ..models.list_health_row import ListHealthRow # noqa: PLC0415
        from ..models.sunset_policy import SunsetPolicy # noqa: PLC0415
        from ..models.sunset_this_week import SunsetThisWeek # noqa: PLC0415
        d = dict(src_dict)
        policy = SunsetPolicy.from_dict(d.pop("policy"))




        policy_week = d.pop("policyWeek")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = ListHealthRow.from_dict(rows_item_data)



            rows.append(rows_item)


        this_week = SunsetThisWeek.from_dict(d.pop("thisWeek"))




        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = HealthTierCard.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        total = d.pop("total")

        trend = []
        _trend = d.pop("trend")
        for trend_item_data in (_trend):
            trend_item = HealthTrendWeek.from_dict(trend_item_data)



            trend.append(trend_item)


        _computed_at = d.pop("computedAt", UNSET)
        computed_at: datetime.datetime | Unset
        if isinstance(_computed_at,  Unset):
            computed_at = UNSET
        else:
            computed_at = datetime.datetime.fromisoformat(_computed_at)




        _insight = d.pop("insight", UNSET)
        insight: ListHealthInsight | Unset
        if isinstance(_insight,  Unset):
            insight = UNSET
        else:
            insight = ListHealthInsight.from_dict(_insight)




        reengage_flow_id = d.pop("reengageFlowId", UNSET)

        win_back_rate = d.pop("winBackRate", UNSET)

        list_health = cls(
            policy=policy,
            policy_week=policy_week,
            rows=rows,
            this_week=this_week,
            tiers=tiers,
            total=total,
            trend=trend,
            computed_at=computed_at,
            insight=insight,
            reengage_flow_id=reengage_flow_id,
            win_back_rate=win_back_rate,
        )

        return list_health

