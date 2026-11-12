from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.email_template_input_kind import EmailTemplateInputKind
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="EmailTemplateInput")



@_attrs_define
class EmailTemplateInput:
    """ 
        Attributes:
            mjml (str):
            name (str):
            description (str | Unset):
            html (str | Unset):
            kind (EmailTemplateInputKind | Unset):  Default: EmailTemplateInputKind.TEMPLATE.
            source_campaign_id (UUID | Unset):
     """

    mjml: str
    name: str
    description: str | Unset = UNSET
    html: str | Unset = UNSET
    kind: EmailTemplateInputKind | Unset = EmailTemplateInputKind.TEMPLATE
    source_campaign_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mjml = self.mjml

        name = self.name

        description = self.description

        html = self.html

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value


        source_campaign_id: str | Unset = UNSET
        if not isinstance(self.source_campaign_id, Unset):
            source_campaign_id = str(self.source_campaign_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mjml": mjml,
            "name": name,
        })
        if description is not UNSET:
            field_dict["description"] = description
        if html is not UNSET:
            field_dict["html"] = html
        if kind is not UNSET:
            field_dict["kind"] = kind
        if source_campaign_id is not UNSET:
            field_dict["sourceCampaignId"] = source_campaign_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mjml = d.pop("mjml")

        name = d.pop("name")

        description = d.pop("description", UNSET)

        html = d.pop("html", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: EmailTemplateInputKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = EmailTemplateInputKind(_kind)




        _source_campaign_id = d.pop("sourceCampaignId", UNSET)
        source_campaign_id: UUID | Unset
        if isinstance(_source_campaign_id,  Unset):
            source_campaign_id = UNSET
        else:
            source_campaign_id = UUID(_source_campaign_id)




        email_template_input = cls(
            mjml=mjml,
            name=name,
            description=description,
            html=html,
            kind=kind,
            source_campaign_id=source_campaign_id,
        )

        return email_template_input

