from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.product_automation_numbers import ProductAutomationNumbers





T = TypeVar("T", bound="ProductNumbers")



@_attrs_define
class ProductNumbers:
    """ 
        Attributes:
            contacts (int):
            from_default (int):
            from_page (int):
            from_shop (int):
            gross_cents (int):
            joined (int):
            none (int):
            offer_list (str):
            orders (int):
            refunded_cents (int):
            refunds (int):
            since (datetime.datetime):
            test (bool):
            automation (ProductAutomationNumbers | Unset):
     """

    contacts: int
    from_default: int
    from_page: int
    from_shop: int
    gross_cents: int
    joined: int
    none: int
    offer_list: str
    orders: int
    refunded_cents: int
    refunds: int
    since: datetime.datetime
    test: bool
    automation: ProductAutomationNumbers | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_automation_numbers import ProductAutomationNumbers # noqa: PLC0415
        contacts = self.contacts

        from_default = self.from_default

        from_page = self.from_page

        from_shop = self.from_shop

        gross_cents = self.gross_cents

        joined = self.joined

        none = self.none

        offer_list = self.offer_list

        orders = self.orders

        refunded_cents = self.refunded_cents

        refunds = self.refunds

        since = self.since.isoformat()

        test = self.test

        automation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.automation, Unset):
            automation = self.automation.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "fromDefault": from_default,
            "fromPage": from_page,
            "fromShop": from_shop,
            "grossCents": gross_cents,
            "joined": joined,
            "none": none,
            "offerList": offer_list,
            "orders": orders,
            "refundedCents": refunded_cents,
            "refunds": refunds,
            "since": since,
            "test": test,
        })
        if automation is not UNSET:
            field_dict["automation"] = automation

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_automation_numbers import ProductAutomationNumbers # noqa: PLC0415
        d = dict(src_dict)
        contacts = d.pop("contacts")

        from_default = d.pop("fromDefault")

        from_page = d.pop("fromPage")

        from_shop = d.pop("fromShop")

        gross_cents = d.pop("grossCents")

        joined = d.pop("joined")

        none = d.pop("none")

        offer_list = d.pop("offerList")

        orders = d.pop("orders")

        refunded_cents = d.pop("refundedCents")

        refunds = d.pop("refunds")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        test = d.pop("test")

        _automation = d.pop("automation", UNSET)
        automation: ProductAutomationNumbers | Unset
        if isinstance(_automation,  Unset):
            automation = UNSET
        else:
            automation = ProductAutomationNumbers.from_dict(_automation)




        product_numbers = cls(
            contacts=contacts,
            from_default=from_default,
            from_page=from_page,
            from_shop=from_shop,
            gross_cents=gross_cents,
            joined=joined,
            none=none,
            offer_list=offer_list,
            orders=orders,
            refunded_cents=refunded_cents,
            refunds=refunds,
            since=since,
            test=test,
            automation=automation,
        )

        return product_numbers

