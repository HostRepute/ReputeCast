from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_address_check_result import ContactAddressCheckResult
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ContactAddressCheck")



@_attrs_define
class ContactAddressCheck:
    """ 
        Attributes:
            can_verify (bool):
            checked (bool):
            connected (bool):
            email (str):
            helpers (list[str]):
            price (float):
            suppressed (bool):
            balance (int | Unset):
            checked_at (datetime.datetime | Unset):
            code (str | Unset):
            reason (str | Unset):
            recheck_from (datetime.datetime | Unset):
            region_label (str | Unset):
            result (ContactAddressCheckResult | Unset):
            service (str | Unset):
            service_name (str | Unset):
            suppressed_at (datetime.datetime | Unset):
            tag (str | Unset):
            verify_service (str | Unset):
            verify_service_name (str | Unset):
     """

    can_verify: bool
    checked: bool
    connected: bool
    email: str
    helpers: list[str]
    price: float
    suppressed: bool
    balance: int | Unset = UNSET
    checked_at: datetime.datetime | Unset = UNSET
    code: str | Unset = UNSET
    reason: str | Unset = UNSET
    recheck_from: datetime.datetime | Unset = UNSET
    region_label: str | Unset = UNSET
    result: ContactAddressCheckResult | Unset = UNSET
    service: str | Unset = UNSET
    service_name: str | Unset = UNSET
    suppressed_at: datetime.datetime | Unset = UNSET
    tag: str | Unset = UNSET
    verify_service: str | Unset = UNSET
    verify_service_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        can_verify = self.can_verify

        checked = self.checked

        connected = self.connected

        email = self.email

        helpers = self.helpers



        price = self.price

        suppressed = self.suppressed

        balance = self.balance

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        code = self.code

        reason = self.reason

        recheck_from: str | Unset = UNSET
        if not isinstance(self.recheck_from, Unset):
            recheck_from = self.recheck_from.isoformat()

        region_label = self.region_label

        result: str | Unset = UNSET
        if not isinstance(self.result, Unset):
            result = self.result.value


        service = self.service

        service_name = self.service_name

        suppressed_at: str | Unset = UNSET
        if not isinstance(self.suppressed_at, Unset):
            suppressed_at = self.suppressed_at.isoformat()

        tag = self.tag

        verify_service = self.verify_service

        verify_service_name = self.verify_service_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canVerify": can_verify,
            "checked": checked,
            "connected": connected,
            "email": email,
            "helpers": helpers,
            "price": price,
            "suppressed": suppressed,
        })
        if balance is not UNSET:
            field_dict["balance"] = balance
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if code is not UNSET:
            field_dict["code"] = code
        if reason is not UNSET:
            field_dict["reason"] = reason
        if recheck_from is not UNSET:
            field_dict["recheckFrom"] = recheck_from
        if region_label is not UNSET:
            field_dict["regionLabel"] = region_label
        if result is not UNSET:
            field_dict["result"] = result
        if service is not UNSET:
            field_dict["service"] = service
        if service_name is not UNSET:
            field_dict["serviceName"] = service_name
        if suppressed_at is not UNSET:
            field_dict["suppressedAt"] = suppressed_at
        if tag is not UNSET:
            field_dict["tag"] = tag
        if verify_service is not UNSET:
            field_dict["verifyService"] = verify_service
        if verify_service_name is not UNSET:
            field_dict["verifyServiceName"] = verify_service_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_verify = d.pop("canVerify")

        checked = d.pop("checked")

        connected = d.pop("connected")

        email = d.pop("email")

        helpers = cast(list[str], d.pop("helpers"))


        price = d.pop("price")

        suppressed = d.pop("suppressed")

        balance = d.pop("balance", UNSET)

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        code = d.pop("code", UNSET)

        reason = d.pop("reason", UNSET)

        _recheck_from = d.pop("recheckFrom", UNSET)
        recheck_from: datetime.datetime | Unset
        if isinstance(_recheck_from,  Unset):
            recheck_from = UNSET
        else:
            recheck_from = datetime.datetime.fromisoformat(_recheck_from)




        region_label = d.pop("regionLabel", UNSET)

        _result = d.pop("result", UNSET)
        result: ContactAddressCheckResult | Unset
        if isinstance(_result,  Unset):
            result = UNSET
        else:
            result = ContactAddressCheckResult(_result)




        service = d.pop("service", UNSET)

        service_name = d.pop("serviceName", UNSET)

        _suppressed_at = d.pop("suppressedAt", UNSET)
        suppressed_at: datetime.datetime | Unset
        if isinstance(_suppressed_at,  Unset):
            suppressed_at = UNSET
        else:
            suppressed_at = datetime.datetime.fromisoformat(_suppressed_at)




        tag = d.pop("tag", UNSET)

        verify_service = d.pop("verifyService", UNSET)

        verify_service_name = d.pop("verifyServiceName", UNSET)

        contact_address_check = cls(
            can_verify=can_verify,
            checked=checked,
            connected=connected,
            email=email,
            helpers=helpers,
            price=price,
            suppressed=suppressed,
            balance=balance,
            checked_at=checked_at,
            code=code,
            reason=reason,
            recheck_from=recheck_from,
            region_label=region_label,
            result=result,
            service=service,
            service_name=service_name,
            suppressed_at=suppressed_at,
            tag=tag,
            verify_service=verify_service,
            verify_service_name=verify_service_name,
        )

        return contact_address_check

