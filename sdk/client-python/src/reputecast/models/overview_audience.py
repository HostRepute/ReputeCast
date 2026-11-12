from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.overview_audience_point import OverviewAudiencePoint
  from ..models.overview_list_growth import OverviewListGrowth
  from ..models.overview_tier import OverviewTier





T = TypeVar("T", bound="OverviewAudience")



@_attrs_define
class OverviewAudience:
    """ 
        Attributes:
            days (int):
            lists (list[OverviewListGrowth]):
            net (int):
            policy (str):
            policy_enabled (bool):
            series (list[OverviewAudiencePoint]):
            subscribers (int):
            tiers (list[OverviewTier]):
            total (int):
            net_percent (float | Unset):
     """

    days: int
    lists: list[OverviewListGrowth]
    net: int
    policy: str
    policy_enabled: bool
    series: list[OverviewAudiencePoint]
    subscribers: int
    tiers: list[OverviewTier]
    total: int
    net_percent: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_audience_point import OverviewAudiencePoint # noqa: PLC0415
        from ..models.overview_list_growth import OverviewListGrowth # noqa: PLC0415
        from ..models.overview_tier import OverviewTier # noqa: PLC0415
        days = self.days

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        net = self.net

        policy = self.policy

        policy_enabled = self.policy_enabled

        series = []
        for series_item_data in self.series:
            series_item = series_item_data.to_dict()
            series.append(series_item)



        subscribers = self.subscribers

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        total = self.total

        net_percent = self.net_percent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "lists": lists,
            "net": net,
            "policy": policy,
            "policyEnabled": policy_enabled,
            "series": series,
            "subscribers": subscribers,
            "tiers": tiers,
            "total": total,
        })
        if net_percent is not UNSET:
            field_dict["netPercent"] = net_percent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_audience_point import OverviewAudiencePoint # noqa: PLC0415
        from ..models.overview_list_growth import OverviewListGrowth # noqa: PLC0415
        from ..models.overview_tier import OverviewTier # noqa: PLC0415
        d = dict(src_dict)
        days = d.pop("days")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = OverviewListGrowth.from_dict(lists_item_data)



            lists.append(lists_item)


        net = d.pop("net")

        policy = d.pop("policy")

        policy_enabled = d.pop("policyEnabled")

        series = []
        _series = d.pop("series")
        for series_item_data in (_series):
            series_item = OverviewAudiencePoint.from_dict(series_item_data)



            series.append(series_item)


        subscribers = d.pop("subscribers")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = OverviewTier.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        total = d.pop("total")

        net_percent = d.pop("netPercent", UNSET)

        overview_audience = cls(
            days=days,
            lists=lists,
            net=net,
            policy=policy,
            policy_enabled=policy_enabled,
            series=series,
            subscribers=subscribers,
            tiers=tiers,
            total=total,
            net_percent=net_percent,
        )

        return overview_audience

