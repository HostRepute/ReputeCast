from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.search_group_kind import SearchGroupKind
from typing import cast

if TYPE_CHECKING:
  from ..models.search_hit import SearchHit





T = TypeVar("T", bound="SearchGroup")



@_attrs_define
class SearchGroup:
    """ 
        Attributes:
            items (list[SearchHit]):
            kind (SearchGroupKind):
     """

    items: list[SearchHit]
    kind: SearchGroupKind





    def to_dict(self) -> dict[str, Any]:
        from ..models.search_hit import SearchHit # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        kind = self.kind.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "kind": kind,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.search_hit import SearchHit # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SearchHit.from_dict(items_item_data)



            items.append(items_item)


        kind = SearchGroupKind(d.pop("kind"))




        search_group = cls(
            items=items,
            kind=kind,
        )

        return search_group

