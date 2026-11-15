from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.brand_voice_content import BrandVoiceContent





T = TypeVar("T", bound="BrandVoice")



@_attrs_define
class BrandVoice:
    """ 
        Attributes:
            content (BrandVoiceContent):
            created_by (str):
            learned (bool):
            revision (int):
            created_at (datetime.datetime | Unset):
     """

    content: BrandVoiceContent
    created_by: str
    learned: bool
    revision: int
    created_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand_voice_content import BrandVoiceContent # noqa: PLC0415
        content = self.content.to_dict()

        created_by = self.created_by

        learned = self.learned

        revision = self.revision

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "content": content,
            "createdBy": created_by,
            "learned": learned,
            "revision": revision,
        })
        if created_at is not UNSET:
            field_dict["createdAt"] = created_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.brand_voice_content import BrandVoiceContent # noqa: PLC0415
        d = dict(src_dict)
        content = BrandVoiceContent.from_dict(d.pop("content"))




        created_by = d.pop("createdBy")

        learned = d.pop("learned")

        revision = d.pop("revision")

        _created_at = d.pop("createdAt", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at,  Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)




        brand_voice = cls(
            content=content,
            created_by=created_by,
            learned=learned,
            revision=revision,
            created_at=created_at,
        )

        return brand_voice

