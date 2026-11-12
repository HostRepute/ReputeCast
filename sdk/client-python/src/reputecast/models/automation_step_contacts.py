from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_step_contact import AutomationStepContact





T = TypeVar("T", bound="AutomationStepContacts")



@_attrs_define
class AutomationStepContacts:
    """ 
        Attributes:
            items (list[AutomationStepContact]):
            total (int):
            next_ (str | Unset):
     """

    items: list[AutomationStepContact]
    total: int
    next_: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_step_contact import AutomationStepContact # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total

        next_ = self.next_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "total": total,
        })
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_step_contact import AutomationStepContact # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = AutomationStepContact.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        next_ = d.pop("next", UNSET)

        automation_step_contacts = cls(
            items=items,
            total=total,
            next_=next_,
        )

        return automation_step_contacts

