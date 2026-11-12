from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.integration_call_view import IntegrationCallView





T = TypeVar("T", bound="IntegrationCallList")



@_attrs_define
class IntegrationCallList:
    """ 
        Attributes:
            errors (int): Errors in the window
            items (list[IntegrationCallView]):
            limit (int):
            offset (int):
            total (int): Calls in the window
     """

    errors: int
    items: list[IntegrationCallView]
    limit: int
    offset: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_call_view import IntegrationCallView # noqa: PLC0415
        errors = self.errors

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        limit = self.limit

        offset = self.offset

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "errors": errors,
            "items": items,
            "limit": limit,
            "offset": offset,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_call_view import IntegrationCallView # noqa: PLC0415
        d = dict(src_dict)
        errors = d.pop("errors")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = IntegrationCallView.from_dict(items_item_data)



            items.append(items_item)


        limit = d.pop("limit")

        offset = d.pop("offset")

        total = d.pop("total")

        integration_call_list = cls(
            errors=errors,
            items=items,
            limit=limit,
            offset=offset,
            total=total,
        )

        return integration_call_list

