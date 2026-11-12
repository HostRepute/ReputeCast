from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ImportVerification")



@_attrs_define
class ImportVerification:
    """ 
        Attributes:
            checked (int):
            cost (float):
            fewer (int):
            invalid (int):
            job_id (str):
            risky (int):
            service (str):
            service_name (str):
            status (str):
            checked_at (datetime.datetime | Unset):
     """

    checked: int
    cost: float
    fewer: int
    invalid: int
    job_id: str
    risky: int
    service: str
    service_name: str
    status: str
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checked = self.checked

        cost = self.cost

        fewer = self.fewer

        invalid = self.invalid

        job_id = self.job_id

        risky = self.risky

        service = self.service

        service_name = self.service_name

        status = self.status

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checked": checked,
            "cost": cost,
            "fewer": fewer,
            "invalid": invalid,
            "jobId": job_id,
            "risky": risky,
            "service": service,
            "serviceName": service_name,
            "status": status,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checked = d.pop("checked")

        cost = d.pop("cost")

        fewer = d.pop("fewer")

        invalid = d.pop("invalid")

        job_id = d.pop("jobId")

        risky = d.pop("risky")

        service = d.pop("service")

        service_name = d.pop("serviceName")

        status = d.pop("status")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        import_verification = cls(
            checked=checked,
            cost=cost,
            fewer=fewer,
            invalid=invalid,
            job_id=job_id,
            risky=risky,
            service=service,
            service_name=service_name,
            status=status,
            checked_at=checked_at,
        )

        return import_verification

