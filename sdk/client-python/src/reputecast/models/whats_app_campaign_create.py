from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="WhatsAppCampaignCreate")



@_attrs_define
class WhatsAppCampaignCreate:
    """ 
        Attributes:
            name (str):
            lists (list[str] | Unset):
            template_id (str | Unset):
     """

    name: str
    lists: list[str] | Unset = UNSET
    template_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        lists: list[str] | Unset = UNSET
        if not isinstance(self.lists, Unset):
            lists = self.lists



        template_id = self.template_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if lists is not UNSET:
            field_dict["lists"] = lists
        if template_id is not UNSET:
            field_dict["templateId"] = template_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        lists = cast(list[str], d.pop("lists", UNSET))


        template_id = d.pop("templateId", UNSET)

        whats_app_campaign_create = cls(
            name=name,
            lists=lists,
            template_id=template_id,
        )

        return whats_app_campaign_create

