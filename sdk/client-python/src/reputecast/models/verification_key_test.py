from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_key_test_kind import VerificationKeyTestKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="VerificationKeyTest")



@_attrs_define
class VerificationKeyTest:
    """ 
        Attributes:
            at (datetime.datetime):
            ok (bool):
            balance (int | Unset):
            error (datetime.datetime | Unset):
            kind (VerificationKeyTestKind | Unset):
            problem (str | Unset):
            status (int | Unset): The service's HTTP status when it refused
     """

    at: datetime.datetime
    ok: bool
    balance: int | Unset = UNSET
    error: datetime.datetime | Unset = UNSET
    kind: VerificationKeyTestKind | Unset = UNSET
    problem: str | Unset = UNSET
    status: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        ok = self.ok

        balance = self.balance

        error: str | Unset = UNSET
        if not isinstance(self.error, Unset):
            error = self.error.isoformat()

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value


        problem = self.problem

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "ok": ok,
        })
        if balance is not UNSET:
            field_dict["balance"] = balance
        if error is not UNSET:
            field_dict["error"] = error
        if kind is not UNSET:
            field_dict["kind"] = kind
        if problem is not UNSET:
            field_dict["problem"] = problem
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        ok = d.pop("ok")

        balance = d.pop("balance", UNSET)

        _error = d.pop("error", UNSET)
        error: datetime.datetime | Unset
        if isinstance(_error,  Unset):
            error = UNSET
        else:
            error = datetime.datetime.fromisoformat(_error)




        _kind = d.pop("kind", UNSET)
        kind: VerificationKeyTestKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = VerificationKeyTestKind(_kind)




        problem = d.pop("problem", UNSET)

        status = d.pop("status", UNSET)

        verification_key_test = cls(
            at=at,
            ok=ok,
            balance=balance,
            error=error,
            kind=kind,
            problem=problem,
            status=status,
        )

        return verification_key_test

