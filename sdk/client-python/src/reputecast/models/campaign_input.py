from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="CampaignInput")



@_attrs_define
class CampaignInput:
    """ 
        Attributes:
            campaign_id (UUID | Unset):
     """

    campaign_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        campaign_id: str | Unset = UNSET
        if not isinstance(self.campaign_id, Unset):
            campaign_id = str(self.campaign_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _campaign_id = d.pop("campaignId", UNSET)
        campaign_id: UUID | Unset
        if isinstance(_campaign_id,  Unset):
            campaign_id = UNSET
        else:
            campaign_id = UUID(_campaign_id)




        campaign_input = cls(
            campaign_id=campaign_id,
        )

        return campaign_input

