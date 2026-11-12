from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_version import TransactionalVersion





T = TypeVar("T", bound="TransactionalVersionList")



@_attrs_define
class TransactionalVersionList:
    """ 
        Attributes:
            can_publish (bool):
            items (list[TransactionalVersion]):
     """

    can_publish: bool
    items: list[TransactionalVersion]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_version import TransactionalVersion # noqa: PLC0415
        can_publish = self.can_publish

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canPublish": can_publish,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_version import TransactionalVersion # noqa: PLC0415
        d = dict(src_dict)
        can_publish = d.pop("canPublish")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = TransactionalVersion.from_dict(items_item_data)



            items.append(items_item)


        transactional_version_list = cls(
            can_publish=can_publish,
            items=items,
        )

        return transactional_version_list

