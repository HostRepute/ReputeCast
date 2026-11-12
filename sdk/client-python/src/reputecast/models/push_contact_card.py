from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.push_contact_subscription import PushContactSubscription





T = TypeVar("T", bound="PushContactCard")



@_attrs_define
class PushContactCard:
    """ 
        Attributes:
            can_edit (bool):
            push_on (bool):
            subscriptions (list[PushContactSubscription]):
     """

    can_edit: bool
    push_on: bool
    subscriptions: list[PushContactSubscription]





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_contact_subscription import PushContactSubscription # noqa: PLC0415
        can_edit = self.can_edit

        push_on = self.push_on

        subscriptions = []
        for subscriptions_item_data in self.subscriptions:
            subscriptions_item = subscriptions_item_data.to_dict()
            subscriptions.append(subscriptions_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "pushOn": push_on,
            "subscriptions": subscriptions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_contact_subscription import PushContactSubscription # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        push_on = d.pop("pushOn")

        subscriptions = []
        _subscriptions = d.pop("subscriptions")
        for subscriptions_item_data in (_subscriptions):
            subscriptions_item = PushContactSubscription.from_dict(subscriptions_item_data)



            subscriptions.append(subscriptions_item)


        push_contact_card = cls(
            can_edit=can_edit,
            push_on=push_on,
            subscriptions=subscriptions,
        )

        return push_contact_card

