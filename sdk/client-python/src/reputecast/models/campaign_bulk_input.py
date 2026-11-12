from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.campaign_bulk_input_action import CampaignBulkInputAction
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="CampaignBulkInput")



@_attrs_define
class CampaignBulkInput:
    """ 
        Attributes:
            action (CampaignBulkInputAction):
            ids (list[str]):
            tag (str | Unset):
     """

    action: CampaignBulkInputAction
    ids: list[str]
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        ids = self.ids



        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "ids": ids,
        })
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = CampaignBulkInputAction(d.pop("action"))




        ids = cast(list[str], d.pop("ids"))


        tag = d.pop("tag", UNSET)

        campaign_bulk_input = cls(
            action=action,
            ids=ids,
            tag=tag,
        )

        return campaign_bulk_input

