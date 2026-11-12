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
  from ..models.verification_service_choice import VerificationServiceChoice





T = TypeVar("T", bound="VerificationEstimate")



@_attrs_define
class VerificationEstimate:
    """ 
        Attributes:
            can_manage (bool):
            can_verify (bool):
            connected (bool):
            contacts (int):
            cost (float):
            enough (bool):
            kind (str):
            label (str):
            manage_helpers (list[str]):
            price (float):
            services (list[VerificationServiceChoice]):
            skip_recent (bool):
            skipped (int):
            to_check (int):
            verify_helpers (list[str]):
            after (int | Unset):
            balance (int | Unset):
            balance_at (datetime.datetime | Unset):
            region_label (str | Unset):
            service (str | Unset):
            service_name (str | Unset):
            skipped_on (datetime.datetime | Unset):
            skipped_service (str | Unset):
     """

    can_manage: bool
    can_verify: bool
    connected: bool
    contacts: int
    cost: float
    enough: bool
    kind: str
    label: str
    manage_helpers: list[str]
    price: float
    services: list[VerificationServiceChoice]
    skip_recent: bool
    skipped: int
    to_check: int
    verify_helpers: list[str]
    after: int | Unset = UNSET
    balance: int | Unset = UNSET
    balance_at: datetime.datetime | Unset = UNSET
    region_label: str | Unset = UNSET
    service: str | Unset = UNSET
    service_name: str | Unset = UNSET
    skipped_on: datetime.datetime | Unset = UNSET
    skipped_service: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_service_choice import VerificationServiceChoice # noqa: PLC0415
        can_manage = self.can_manage

        can_verify = self.can_verify

        connected = self.connected

        contacts = self.contacts

        cost = self.cost

        enough = self.enough

        kind = self.kind

        label = self.label

        manage_helpers = self.manage_helpers



        price = self.price

        services = []
        for services_item_data in self.services:
            services_item = services_item_data.to_dict()
            services.append(services_item)



        skip_recent = self.skip_recent

        skipped = self.skipped

        to_check = self.to_check

        verify_helpers = self.verify_helpers



        after = self.after

        balance = self.balance

        balance_at: str | Unset = UNSET
        if not isinstance(self.balance_at, Unset):
            balance_at = self.balance_at.isoformat()

        region_label = self.region_label

        service = self.service

        service_name = self.service_name

        skipped_on: str | Unset = UNSET
        if not isinstance(self.skipped_on, Unset):
            skipped_on = self.skipped_on.isoformat()

        skipped_service = self.skipped_service


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "canVerify": can_verify,
            "connected": connected,
            "contacts": contacts,
            "cost": cost,
            "enough": enough,
            "kind": kind,
            "label": label,
            "manageHelpers": manage_helpers,
            "price": price,
            "services": services,
            "skipRecent": skip_recent,
            "skipped": skipped,
            "toCheck": to_check,
            "verifyHelpers": verify_helpers,
        })
        if after is not UNSET:
            field_dict["after"] = after
        if balance is not UNSET:
            field_dict["balance"] = balance
        if balance_at is not UNSET:
            field_dict["balanceAt"] = balance_at
        if region_label is not UNSET:
            field_dict["regionLabel"] = region_label
        if service is not UNSET:
            field_dict["service"] = service
        if service_name is not UNSET:
            field_dict["serviceName"] = service_name
        if skipped_on is not UNSET:
            field_dict["skippedOn"] = skipped_on
        if skipped_service is not UNSET:
            field_dict["skippedService"] = skipped_service

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.verification_service_choice import VerificationServiceChoice # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        can_verify = d.pop("canVerify")

        connected = d.pop("connected")

        contacts = d.pop("contacts")

        cost = d.pop("cost")

        enough = d.pop("enough")

        kind = d.pop("kind")

        label = d.pop("label")

        manage_helpers = cast(list[str], d.pop("manageHelpers"))


        price = d.pop("price")

        services = []
        _services = d.pop("services")
        for services_item_data in (_services):
            services_item = VerificationServiceChoice.from_dict(services_item_data)



            services.append(services_item)


        skip_recent = d.pop("skipRecent")

        skipped = d.pop("skipped")

        to_check = d.pop("toCheck")

        verify_helpers = cast(list[str], d.pop("verifyHelpers"))


        after = d.pop("after", UNSET)

        balance = d.pop("balance", UNSET)

        _balance_at = d.pop("balanceAt", UNSET)
        balance_at: datetime.datetime | Unset
        if isinstance(_balance_at,  Unset):
            balance_at = UNSET
        else:
            balance_at = datetime.datetime.fromisoformat(_balance_at)




        region_label = d.pop("regionLabel", UNSET)

        service = d.pop("service", UNSET)

        service_name = d.pop("serviceName", UNSET)

        _skipped_on = d.pop("skippedOn", UNSET)
        skipped_on: datetime.datetime | Unset
        if isinstance(_skipped_on,  Unset):
            skipped_on = UNSET
        else:
            skipped_on = datetime.datetime.fromisoformat(_skipped_on)




        skipped_service = d.pop("skippedService", UNSET)

        verification_estimate = cls(
            can_manage=can_manage,
            can_verify=can_verify,
            connected=connected,
            contacts=contacts,
            cost=cost,
            enough=enough,
            kind=kind,
            label=label,
            manage_helpers=manage_helpers,
            price=price,
            services=services,
            skip_recent=skip_recent,
            skipped=skipped,
            to_check=to_check,
            verify_helpers=verify_helpers,
            after=after,
            balance=balance,
            balance_at=balance_at,
            region_label=region_label,
            service=service,
            service_name=service_name,
            skipped_on=skipped_on,
            skipped_service=skipped_service,
        )

        return verification_estimate

