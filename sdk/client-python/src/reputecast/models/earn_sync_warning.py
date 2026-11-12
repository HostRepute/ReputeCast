from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.earn_member_row import EarnMemberRow





T = TypeVar("T", bound="EarnSyncWarning")



@_attrs_define
class EarnSyncWarning:
    """ 
        Attributes:
            subscriptions (list[EarnMemberRow]):
     """

    subscriptions: list[EarnMemberRow]





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        subscriptions = []
        for subscriptions_item_data in self.subscriptions:
            subscriptions_item = subscriptions_item_data.to_dict()
            subscriptions.append(subscriptions_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "subscriptions": subscriptions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        d = dict(src_dict)
        subscriptions = []
        _subscriptions = d.pop("subscriptions")
        for subscriptions_item_data in (_subscriptions):
            subscriptions_item = EarnMemberRow.from_dict(subscriptions_item_data)



            subscriptions.append(subscriptions_item)


        earn_sync_warning = cls(
            subscriptions=subscriptions,
        )

        return earn_sync_warning

