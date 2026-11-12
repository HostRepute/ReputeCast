from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.render import Render





T = TypeVar("T", bound="RenderList")



@_attrs_define
class RenderList:
    """ 
        Attributes:
            items (list[Render]):
     """

    items: list[Render]





    def to_dict(self) -> dict[str, Any]:
        from ..models.render import Render # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.render import Render # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = Render.from_dict(items_item_data)



            items.append(items_item)


        render_list = cls(
            items=items,
        )

        return render_list

