from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.email_template_detail_kind import EmailTemplateDetailKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.mjml_thumb_item import MJMLThumbItem





T = TypeVar("T", bound="EmailTemplateDetail")



@_attrs_define
class EmailTemplateDetail:
    """ 
        Attributes:
            builtin (bool):
            description (str):
            html (str):
            id (str):  Example: starter:weekly-digest.
            kind (EmailTemplateDetailKind):
            mjml (str):
            name (str):
            size_bytes (int):
            thumb (list[MJMLThumbItem]):
            source_campaign_id (str | Unset):
            updated_at (datetime.datetime | Unset):
            updated_by (str | Unset):
     """

    builtin: bool
    description: str
    html: str
    id: str
    kind: EmailTemplateDetailKind
    mjml: str
    name: str
    size_bytes: int
    thumb: list[MJMLThumbItem]
    source_campaign_id: str | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.mjml_thumb_item import MJMLThumbItem # noqa: PLC0415
        builtin = self.builtin

        description = self.description

        html = self.html

        id = self.id

        kind = self.kind.value

        mjml = self.mjml

        name = self.name

        size_bytes = self.size_bytes

        thumb = []
        for thumb_item_data in self.thumb:
            thumb_item = thumb_item_data.to_dict()
            thumb.append(thumb_item)



        source_campaign_id = self.source_campaign_id

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "builtin": builtin,
            "description": description,
            "html": html,
            "id": id,
            "kind": kind,
            "mjml": mjml,
            "name": name,
            "sizeBytes": size_bytes,
            "thumb": thumb,
        })
        if source_campaign_id is not UNSET:
            field_dict["sourceCampaignId"] = source_campaign_id
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mjml_thumb_item import MJMLThumbItem # noqa: PLC0415
        d = dict(src_dict)
        builtin = d.pop("builtin")

        description = d.pop("description")

        html = d.pop("html")

        id = d.pop("id")

        kind = EmailTemplateDetailKind(d.pop("kind"))




        mjml = d.pop("mjml")

        name = d.pop("name")

        size_bytes = d.pop("sizeBytes")

        thumb = []
        _thumb = d.pop("thumb")
        for thumb_item_data in (_thumb):
            thumb_item = MJMLThumbItem.from_dict(thumb_item_data)



            thumb.append(thumb_item)


        source_campaign_id = d.pop("sourceCampaignId", UNSET)

        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at,  Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)




        updated_by = d.pop("updatedBy", UNSET)

        email_template_detail = cls(
            builtin=builtin,
            description=description,
            html=html,
            id=id,
            kind=kind,
            mjml=mjml,
            name=name,
            size_bytes=size_bytes,
            thumb=thumb,
            source_campaign_id=source_campaign_id,
            updated_at=updated_at,
            updated_by=updated_by,
        )

        return email_template_detail

