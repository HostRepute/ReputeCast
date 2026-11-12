from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_diff_item import AutomationDiffItem





T = TypeVar("T", bound="AutomationDiff")



@_attrs_define
class AutomationDiff:
    """ 
        Attributes:
            changes (int):
            from_ (int):
            items (list[AutomationDiffItem]):
            to (int):
     """

    changes: int
    from_: int
    items: list[AutomationDiffItem]
    to: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_diff_item import AutomationDiffItem # noqa: PLC0415
        changes = self.changes

        from_ = self.from_

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "from": from_,
            "items": items,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_diff_item import AutomationDiffItem # noqa: PLC0415
        d = dict(src_dict)
        changes = d.pop("changes")

        from_ = d.pop("from")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = AutomationDiffItem.from_dict(items_item_data)



            items.append(items_item)


        to = d.pop("to")

        automation_diff = cls(
            changes=changes,
            from_=from_,
            items=items,
            to=to,
        )

        return automation_diff

