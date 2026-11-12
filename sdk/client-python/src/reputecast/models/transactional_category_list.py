from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_category import TransactionalCategory





T = TypeVar("T", bound="TransactionalCategoryList")



@_attrs_define
class TransactionalCategoryList:
    """ 
        Attributes:
            can_edit (bool):
            items (list[TransactionalCategory]):
            reason (str | Unset):
     """

    can_edit: bool
    items: list[TransactionalCategory]
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_category import TransactionalCategory # noqa: PLC0415
        can_edit = self.can_edit

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "items": items,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_category import TransactionalCategory # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = TransactionalCategory.from_dict(items_item_data)



            items.append(items_item)


        reason = d.pop("reason", UNSET)

        transactional_category_list = cls(
            can_edit=can_edit,
            items=items,
            reason=reason,
        )

        return transactional_category_list

