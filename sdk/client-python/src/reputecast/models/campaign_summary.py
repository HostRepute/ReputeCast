from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.provider_cost import ProviderCost





T = TypeVar("T", bound="CampaignSummary")



@_attrs_define
class CampaignSummary:
    """ 
        Attributes:
            campaigns (int):
            cost (float): Dollars
            cost_by (list[ProviderCost]):
            days (int):
            sent (int):
            bounces (float | Unset):
            click_to_open (float | Unset):
            clicks (float | Unset):
            complaints (float | Unset):
            delivered (float | Unset):
            human_opens (float | Unset):
     """

    campaigns: int
    cost: float
    cost_by: list[ProviderCost]
    days: int
    sent: int
    bounces: float | Unset = UNSET
    click_to_open: float | Unset = UNSET
    clicks: float | Unset = UNSET
    complaints: float | Unset = UNSET
    delivered: float | Unset = UNSET
    human_opens: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_cost import ProviderCost # noqa: PLC0415
        campaigns = self.campaigns

        cost = self.cost

        cost_by = []
        for cost_by_item_data in self.cost_by:
            cost_by_item = cost_by_item_data.to_dict()
            cost_by.append(cost_by_item)



        days = self.days

        sent = self.sent

        bounces = self.bounces

        click_to_open = self.click_to_open

        clicks = self.clicks

        complaints = self.complaints

        delivered = self.delivered

        human_opens = self.human_opens


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaigns": campaigns,
            "cost": cost,
            "costBy": cost_by,
            "days": days,
            "sent": sent,
        })
        if bounces is not UNSET:
            field_dict["bounces"] = bounces
        if click_to_open is not UNSET:
            field_dict["clickToOpen"] = click_to_open
        if clicks is not UNSET:
            field_dict["clicks"] = clicks
        if complaints is not UNSET:
            field_dict["complaints"] = complaints
        if delivered is not UNSET:
            field_dict["delivered"] = delivered
        if human_opens is not UNSET:
            field_dict["humanOpens"] = human_opens

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_cost import ProviderCost # noqa: PLC0415
        d = dict(src_dict)
        campaigns = d.pop("campaigns")

        cost = d.pop("cost")

        cost_by = []
        _cost_by = d.pop("costBy")
        for cost_by_item_data in (_cost_by):
            cost_by_item = ProviderCost.from_dict(cost_by_item_data)



            cost_by.append(cost_by_item)


        days = d.pop("days")

        sent = d.pop("sent")

        bounces = d.pop("bounces", UNSET)

        click_to_open = d.pop("clickToOpen", UNSET)

        clicks = d.pop("clicks", UNSET)

        complaints = d.pop("complaints", UNSET)

        delivered = d.pop("delivered", UNSET)

        human_opens = d.pop("humanOpens", UNSET)

        campaign_summary = cls(
            campaigns=campaigns,
            cost=cost,
            cost_by=cost_by,
            days=days,
            sent=sent,
            bounces=bounces,
            click_to_open=click_to_open,
            clicks=clicks,
            complaints=complaints,
            delivered=delivered,
            human_opens=human_opens,
        )

        return campaign_summary

