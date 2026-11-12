from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceSyncFacts")



@_attrs_define
class CommerceSyncFacts:
    """ 
        Attributes:
            accepted (int):
            collections (list[str]):
            customers_from_orders (bool):
            import_minutes (int):
            known (int):
            missed (int):
            months (int):
            new (int):
            not_added (int):
            orders_imported (int):
            products (int):
            webhooks (int):
            webhooks_total (int):
            import_finished (datetime.datetime | Unset):
            reconciled_at (datetime.datetime | Unset):
     """

    accepted: int
    collections: list[str]
    customers_from_orders: bool
    import_minutes: int
    known: int
    missed: int
    months: int
    new: int
    not_added: int
    orders_imported: int
    products: int
    webhooks: int
    webhooks_total: int
    import_finished: datetime.datetime | Unset = UNSET
    reconciled_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accepted = self.accepted

        collections = self.collections



        customers_from_orders = self.customers_from_orders

        import_minutes = self.import_minutes

        known = self.known

        missed = self.missed

        months = self.months

        new = self.new

        not_added = self.not_added

        orders_imported = self.orders_imported

        products = self.products

        webhooks = self.webhooks

        webhooks_total = self.webhooks_total

        import_finished: str | Unset = UNSET
        if not isinstance(self.import_finished, Unset):
            import_finished = self.import_finished.isoformat()

        reconciled_at: str | Unset = UNSET
        if not isinstance(self.reconciled_at, Unset):
            reconciled_at = self.reconciled_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "collections": collections,
            "customersFromOrders": customers_from_orders,
            "importMinutes": import_minutes,
            "known": known,
            "missed": missed,
            "months": months,
            "new": new,
            "notAdded": not_added,
            "ordersImported": orders_imported,
            "products": products,
            "webhooks": webhooks,
            "webhooksTotal": webhooks_total,
        })
        if import_finished is not UNSET:
            field_dict["importFinished"] = import_finished
        if reconciled_at is not UNSET:
            field_dict["reconciledAt"] = reconciled_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accepted = d.pop("accepted")

        collections = cast(list[str], d.pop("collections"))


        customers_from_orders = d.pop("customersFromOrders")

        import_minutes = d.pop("importMinutes")

        known = d.pop("known")

        missed = d.pop("missed")

        months = d.pop("months")

        new = d.pop("new")

        not_added = d.pop("notAdded")

        orders_imported = d.pop("ordersImported")

        products = d.pop("products")

        webhooks = d.pop("webhooks")

        webhooks_total = d.pop("webhooksTotal")

        _import_finished = d.pop("importFinished", UNSET)
        import_finished: datetime.datetime | Unset
        if isinstance(_import_finished,  Unset):
            import_finished = UNSET
        else:
            import_finished = datetime.datetime.fromisoformat(_import_finished)




        _reconciled_at = d.pop("reconciledAt", UNSET)
        reconciled_at: datetime.datetime | Unset
        if isinstance(_reconciled_at,  Unset):
            reconciled_at = UNSET
        else:
            reconciled_at = datetime.datetime.fromisoformat(_reconciled_at)




        commerce_sync_facts = cls(
            accepted=accepted,
            collections=collections,
            customers_from_orders=customers_from_orders,
            import_minutes=import_minutes,
            known=known,
            missed=missed,
            months=months,
            new=new,
            not_added=not_added,
            orders_imported=orders_imported,
            products=products,
            webhooks=webhooks,
            webhooks_total=webhooks_total,
            import_finished=import_finished,
            reconciled_at=reconciled_at,
        )

        return commerce_sync_facts

