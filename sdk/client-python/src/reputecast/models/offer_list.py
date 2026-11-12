from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.offer import Offer





T = TypeVar("T", bound="OfferList")



@_attrs_define
class OfferList:
    """ 
        Attributes:
            account (str):
            can_edit (bool):
            connected (bool):
            helpers (list[str]):
            items (list[Offer]):
            provider (str):
     """

    account: str
    can_edit: bool
    connected: bool
    helpers: list[str]
    items: list[Offer]
    provider: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.offer import Offer # noqa: PLC0415
        account = self.account

        can_edit = self.can_edit

        connected = self.connected

        helpers = self.helpers



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        provider = self.provider


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "canEdit": can_edit,
            "connected": connected,
            "helpers": helpers,
            "items": items,
            "provider": provider,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.offer import Offer # noqa: PLC0415
        d = dict(src_dict)
        account = d.pop("account")

        can_edit = d.pop("canEdit")

        connected = d.pop("connected")

        helpers = cast(list[str], d.pop("helpers"))


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = Offer.from_dict(items_item_data)



            items.append(items_item)


        provider = d.pop("provider")

        offer_list = cls(
            account=account,
            can_edit=can_edit,
            connected=connected,
            helpers=helpers,
            items=items,
            provider=provider,
        )

        return offer_list

