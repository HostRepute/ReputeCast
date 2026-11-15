from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.brand_voice import BrandVoice





T = TypeVar("T", bound="BrandVoiceHistory")



@_attrs_define
class BrandVoiceHistory:
    """ 
        Attributes:
            items (list[BrandVoice]):
            before_revision (int | Unset):
     """

    items: list[BrandVoice]
    before_revision: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand_voice import BrandVoice # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        before_revision = self.before_revision


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
        })
        if before_revision is not UNSET:
            field_dict["beforeRevision"] = before_revision

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.brand_voice import BrandVoice # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = BrandVoice.from_dict(items_item_data)



            items.append(items_item)


        before_revision = d.pop("beforeRevision", UNSET)

        brand_voice_history = cls(
            items=items,
            before_revision=before_revision,
        )

        return brand_voice_history

