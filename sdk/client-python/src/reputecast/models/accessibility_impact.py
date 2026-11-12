from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_impact_on_serious import AccessibilityImpactOnSerious
from typing import cast

if TYPE_CHECKING:
  from ..models.accessibility_impact_campaign import AccessibilityImpactCampaign
  from ..models.accessibility_impact_flow import AccessibilityImpactFlow





T = TypeVar("T", bound="AccessibilityImpact")



@_attrs_define
class AccessibilityImpact:
    """ 
        Attributes:
            campaigns (list[AccessibilityImpactCampaign]):
            flows (list[AccessibilityImpactFlow]):
            held (int):
            on_serious (AccessibilityImpactOnSerious):
     """

    campaigns: list[AccessibilityImpactCampaign]
    flows: list[AccessibilityImpactFlow]
    held: int
    on_serious: AccessibilityImpactOnSerious





    def to_dict(self) -> dict[str, Any]:
        from ..models.accessibility_impact_campaign import AccessibilityImpactCampaign # noqa: PLC0415
        from ..models.accessibility_impact_flow import AccessibilityImpactFlow # noqa: PLC0415
        campaigns = []
        for campaigns_item_data in self.campaigns:
            campaigns_item = campaigns_item_data.to_dict()
            campaigns.append(campaigns_item)



        flows = []
        for flows_item_data in self.flows:
            flows_item = flows_item_data.to_dict()
            flows.append(flows_item)



        held = self.held

        on_serious = self.on_serious.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaigns": campaigns,
            "flows": flows,
            "held": held,
            "onSerious": on_serious,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.accessibility_impact_campaign import AccessibilityImpactCampaign # noqa: PLC0415
        from ..models.accessibility_impact_flow import AccessibilityImpactFlow # noqa: PLC0415
        d = dict(src_dict)
        campaigns = []
        _campaigns = d.pop("campaigns")
        for campaigns_item_data in (_campaigns):
            campaigns_item = AccessibilityImpactCampaign.from_dict(campaigns_item_data)



            campaigns.append(campaigns_item)


        flows = []
        _flows = d.pop("flows")
        for flows_item_data in (_flows):
            flows_item = AccessibilityImpactFlow.from_dict(flows_item_data)



            flows.append(flows_item)


        held = d.pop("held")

        on_serious = AccessibilityImpactOnSerious(d.pop("onSerious"))




        accessibility_impact = cls(
            campaigns=campaigns,
            flows=flows,
            held=held,
            on_serious=on_serious,
        )

        return accessibility_impact

