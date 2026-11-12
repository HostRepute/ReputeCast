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
  from ..models.route_campaign_condition import RouteCampaignCondition
  from ..models.route_campaign_draft import RouteCampaignDraft
  from ..models.route_campaign_share import RouteCampaignShare
  from ..models.route_campaign_warmup import RouteCampaignWarmup
  from ..models.route_stream_impact import RouteStreamImpact





T = TypeVar("T", bound="RouteCampaignResult")



@_attrs_define
class RouteCampaignResult:
    """ 
        Attributes:
            campaign_id (str):
            conditions (list[RouteCampaignCondition]):
            domain (str):
            estimated (bool):
            impact (list[RouteStreamImpact]):
            lists (list[str]):
            name (str):
            none (bool):
            shares (list[RouteCampaignShare]):
            simulated (int):
            sunset (int):
            timezone (str):
            total (int):
            warmup (list[RouteCampaignWarmup]):
            draft (RouteCampaignDraft | Unset):
            scheduled_at (datetime.datetime | Unset):
     """

    campaign_id: str
    conditions: list[RouteCampaignCondition]
    domain: str
    estimated: bool
    impact: list[RouteStreamImpact]
    lists: list[str]
    name: str
    none: bool
    shares: list[RouteCampaignShare]
    simulated: int
    sunset: int
    timezone: str
    total: int
    warmup: list[RouteCampaignWarmup]
    draft: RouteCampaignDraft | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.route_campaign_condition import RouteCampaignCondition # noqa: PLC0415
        from ..models.route_campaign_draft import RouteCampaignDraft # noqa: PLC0415
        from ..models.route_campaign_share import RouteCampaignShare # noqa: PLC0415
        from ..models.route_campaign_warmup import RouteCampaignWarmup # noqa: PLC0415
        from ..models.route_stream_impact import RouteStreamImpact # noqa: PLC0415
        campaign_id = self.campaign_id

        conditions = []
        for conditions_item_data in self.conditions:
            conditions_item = conditions_item_data.to_dict()
            conditions.append(conditions_item)



        domain = self.domain

        estimated = self.estimated

        impact = []
        for impact_item_data in self.impact:
            impact_item = impact_item_data.to_dict()
            impact.append(impact_item)



        lists = self.lists



        name = self.name

        none = self.none

        shares = []
        for shares_item_data in self.shares:
            shares_item = shares_item_data.to_dict()
            shares.append(shares_item)



        simulated = self.simulated

        sunset = self.sunset

        timezone = self.timezone

        total = self.total

        warmup = []
        for warmup_item_data in self.warmup:
            warmup_item = warmup_item_data.to_dict()
            warmup.append(warmup_item)



        draft: dict[str, Any] | Unset = UNSET
        if not isinstance(self.draft, Unset):
            draft = self.draft.to_dict()

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "conditions": conditions,
            "domain": domain,
            "estimated": estimated,
            "impact": impact,
            "lists": lists,
            "name": name,
            "none": none,
            "shares": shares,
            "simulated": simulated,
            "sunset": sunset,
            "timezone": timezone,
            "total": total,
            "warmup": warmup,
        })
        if draft is not UNSET:
            field_dict["draft"] = draft
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.route_campaign_condition import RouteCampaignCondition # noqa: PLC0415
        from ..models.route_campaign_draft import RouteCampaignDraft # noqa: PLC0415
        from ..models.route_campaign_share import RouteCampaignShare # noqa: PLC0415
        from ..models.route_campaign_warmup import RouteCampaignWarmup # noqa: PLC0415
        from ..models.route_stream_impact import RouteStreamImpact # noqa: PLC0415
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        conditions = []
        _conditions = d.pop("conditions")
        for conditions_item_data in (_conditions):
            conditions_item = RouteCampaignCondition.from_dict(conditions_item_data)



            conditions.append(conditions_item)


        domain = d.pop("domain")

        estimated = d.pop("estimated")

        impact = []
        _impact = d.pop("impact")
        for impact_item_data in (_impact):
            impact_item = RouteStreamImpact.from_dict(impact_item_data)



            impact.append(impact_item)


        lists = cast(list[str], d.pop("lists"))


        name = d.pop("name")

        none = d.pop("none")

        shares = []
        _shares = d.pop("shares")
        for shares_item_data in (_shares):
            shares_item = RouteCampaignShare.from_dict(shares_item_data)



            shares.append(shares_item)


        simulated = d.pop("simulated")

        sunset = d.pop("sunset")

        timezone = d.pop("timezone")

        total = d.pop("total")

        warmup = []
        _warmup = d.pop("warmup")
        for warmup_item_data in (_warmup):
            warmup_item = RouteCampaignWarmup.from_dict(warmup_item_data)



            warmup.append(warmup_item)


        _draft = d.pop("draft", UNSET)
        draft: RouteCampaignDraft | Unset
        if isinstance(_draft,  Unset):
            draft = UNSET
        else:
            draft = RouteCampaignDraft.from_dict(_draft)




        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        route_campaign_result = cls(
            campaign_id=campaign_id,
            conditions=conditions,
            domain=domain,
            estimated=estimated,
            impact=impact,
            lists=lists,
            name=name,
            none=none,
            shares=shares,
            simulated=simulated,
            sunset=sunset,
            timezone=timezone,
            total=total,
            warmup=warmup,
            draft=draft,
            scheduled_at=scheduled_at,
        )

        return route_campaign_result

