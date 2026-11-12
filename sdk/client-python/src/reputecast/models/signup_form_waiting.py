from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_waiting_item import SignupFormWaitingItem





T = TypeVar("T", bound="SignupFormWaiting")



@_attrs_define
class SignupFormWaiting:
    """ 
        Attributes:
            items (list[SignupFormWaitingItem]):
            system_mail (bool):
            total (int):
     """

    items: list[SignupFormWaitingItem]
    system_mail: bool
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_waiting_item import SignupFormWaitingItem # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        system_mail = self.system_mail

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "systemMail": system_mail,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_waiting_item import SignupFormWaitingItem # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SignupFormWaitingItem.from_dict(items_item_data)



            items.append(items_item)


        system_mail = d.pop("systemMail")

        total = d.pop("total")

        signup_form_waiting = cls(
            items=items,
            system_mail=system_mail,
            total=total,
        )

        return signup_form_waiting

