from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="ComposerCreateCampaign")



@_attrs_define
class ComposerCreateCampaign:
    """ 
        Attributes:
            from_campaign_id (UUID | Unset): Start from a copy of another campaign's content
            name (str | Unset):
            template_id (str | Unset): Start from a template (starter:<key> or a saved template id)
     """

    from_campaign_id: UUID | Unset = UNSET
    name: str | Unset = UNSET
    template_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from_campaign_id: str | Unset = UNSET
        if not isinstance(self.from_campaign_id, Unset):
            from_campaign_id = str(self.from_campaign_id)

        name = self.name

        template_id = self.template_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if from_campaign_id is not UNSET:
            field_dict["fromCampaignId"] = from_campaign_id
        if name is not UNSET:
            field_dict["name"] = name
        if template_id is not UNSET:
            field_dict["templateId"] = template_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _from_campaign_id = d.pop("fromCampaignId", UNSET)
        from_campaign_id: UUID | Unset
        if isinstance(_from_campaign_id,  Unset):
            from_campaign_id = UNSET
        else:
            from_campaign_id = UUID(_from_campaign_id)




        name = d.pop("name", UNSET)

        template_id = d.pop("templateId", UNSET)

        composer_create_campaign = cls(
            from_campaign_id=from_campaign_id,
            name=name,
            template_id=template_id,
        )

        return composer_create_campaign

