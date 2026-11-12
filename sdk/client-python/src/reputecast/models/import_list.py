from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.import_summary import ImportSummary





T = TypeVar("T", bound="ImportList")



@_attrs_define
class ImportList:
    """ 
        Attributes:
            items (list[ImportSummary]):
            active (str | Unset):
     """

    items: list[ImportSummary]
    active: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_summary import ImportSummary # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        active = self.active


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
        })
        if active is not UNSET:
            field_dict["active"] = active

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_summary import ImportSummary # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ImportSummary.from_dict(items_item_data)



            items.append(items_item)


        active = d.pop("active", UNSET)

        import_list = cls(
            items=items,
            active=active,
        )

        return import_list

