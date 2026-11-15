from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
from uuid import UUID

if TYPE_CHECKING:
  from ..models.brand_voice_content import BrandVoiceContent





T = TypeVar("T", bound="BrandVoiceSaveInput")



@_attrs_define
class BrandVoiceSaveInput:
    """ 
        Attributes:
            base_revision (int):
            content (BrandVoiceContent):
            suggestion_id (UUID | Unset):
     """

    base_revision: int
    content: BrandVoiceContent
    suggestion_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand_voice_content import BrandVoiceContent # noqa: PLC0415
        base_revision = self.base_revision

        content = self.content.to_dict()

        suggestion_id: str | Unset = UNSET
        if not isinstance(self.suggestion_id, Unset):
            suggestion_id = str(self.suggestion_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseRevision": base_revision,
            "content": content,
        })
        if suggestion_id is not UNSET:
            field_dict["suggestionId"] = suggestion_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.brand_voice_content import BrandVoiceContent # noqa: PLC0415
        d = dict(src_dict)
        base_revision = d.pop("baseRevision")

        content = BrandVoiceContent.from_dict(d.pop("content"))




        _suggestion_id = d.pop("suggestionId", UNSET)
        suggestion_id: UUID | Unset
        if isinstance(_suggestion_id,  Unset):
            suggestion_id = UNSET
        else:
            suggestion_id = UUID(_suggestion_id)




        brand_voice_save_input = cls(
            base_revision=base_revision,
            content=content,
            suggestion_id=suggestion_id,
        )

        return brand_voice_save_input

