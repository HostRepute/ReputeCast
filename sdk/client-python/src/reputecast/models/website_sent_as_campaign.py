from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteSentAsCampaign")



@_attrs_define
class WebsiteSentAsCampaign:
    """ 
        Attributes:
            campaign_id (str):
     """

    campaign_id: str





    def to_dict(self) -> dict[str, Any]:
        campaign_id = self.campaign_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        website_sent_as_campaign = cls(
            campaign_id=campaign_id,
        )

        return website_sent_as_campaign

