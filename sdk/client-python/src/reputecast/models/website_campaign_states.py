from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.website_campaign_web_state import WebsiteCampaignWebState





T = TypeVar("T", bound="WebsiteCampaignStates")



@_attrs_define
class WebsiteCampaignStates:
    """ 
        Attributes:
            site_on (bool):
            states (list[WebsiteCampaignWebState]):
     """

    site_on: bool
    states: list[WebsiteCampaignWebState]





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_campaign_web_state import WebsiteCampaignWebState # noqa: PLC0415
        site_on = self.site_on

        states = []
        for states_item_data in self.states:
            states_item = states_item_data.to_dict()
            states.append(states_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "siteOn": site_on,
            "states": states,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_campaign_web_state import WebsiteCampaignWebState # noqa: PLC0415
        d = dict(src_dict)
        site_on = d.pop("siteOn")

        states = []
        _states = d.pop("states")
        for states_item_data in (_states):
            states_item = WebsiteCampaignWebState.from_dict(states_item_data)



            states.append(states_item)


        website_campaign_states = cls(
            site_on=site_on,
            states=states,
        )

        return website_campaign_states

