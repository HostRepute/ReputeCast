from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_import_phase import CommerceImportPhase
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.commerce_progress import CommerceProgress





T = TypeVar("T", bound="CommerceImport")



@_attrs_define
class CommerceImport:
    """ 
        Attributes:
            accepted (int):
            collections (int):
            customers (CommerceProgress):
            customers_skipped (bool):
            known (int):
            matching (CommerceProgress):
            minutes_left (int):
            months (int):
            not_added (int):
            orders (CommerceProgress):
            phase (CommerceImportPhase):
            products (CommerceProgress):
            started_at (datetime.datetime):
            started_by (str):
            finished_at (datetime.datetime | Unset):
     """

    accepted: int
    collections: int
    customers: CommerceProgress
    customers_skipped: bool
    known: int
    matching: CommerceProgress
    minutes_left: int
    months: int
    not_added: int
    orders: CommerceProgress
    phase: CommerceImportPhase
    products: CommerceProgress
    started_at: datetime.datetime
    started_by: str
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_progress import CommerceProgress # noqa: PLC0415
        accepted = self.accepted

        collections = self.collections

        customers = self.customers.to_dict()

        customers_skipped = self.customers_skipped

        known = self.known

        matching = self.matching.to_dict()

        minutes_left = self.minutes_left

        months = self.months

        not_added = self.not_added

        orders = self.orders.to_dict()

        phase = self.phase.value

        products = self.products.to_dict()

        started_at = self.started_at.isoformat()

        started_by = self.started_by

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "collections": collections,
            "customers": customers,
            "customersSkipped": customers_skipped,
            "known": known,
            "matching": matching,
            "minutesLeft": minutes_left,
            "months": months,
            "notAdded": not_added,
            "orders": orders,
            "phase": phase,
            "products": products,
            "startedAt": started_at,
            "startedBy": started_by,
        })
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_progress import CommerceProgress # noqa: PLC0415
        d = dict(src_dict)
        accepted = d.pop("accepted")

        collections = d.pop("collections")

        customers = CommerceProgress.from_dict(d.pop("customers"))




        customers_skipped = d.pop("customersSkipped")

        known = d.pop("known")

        matching = CommerceProgress.from_dict(d.pop("matching"))




        minutes_left = d.pop("minutesLeft")

        months = d.pop("months")

        not_added = d.pop("notAdded")

        orders = CommerceProgress.from_dict(d.pop("orders"))




        phase = CommerceImportPhase(d.pop("phase"))




        products = CommerceProgress.from_dict(d.pop("products"))




        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        started_by = d.pop("startedBy")

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        commerce_import = cls(
            accepted=accepted,
            collections=collections,
            customers=customers,
            customers_skipped=customers_skipped,
            known=known,
            matching=matching,
            minutes_left=minutes_left,
            months=months,
            not_added=not_added,
            orders=orders,
            phase=phase,
            products=products,
            started_at=started_at,
            started_by=started_by,
            finished_at=finished_at,
        )

        return commerce_import

