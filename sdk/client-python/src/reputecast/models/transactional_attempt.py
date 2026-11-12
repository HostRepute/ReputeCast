from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalAttempt")



@_attrs_define
class TransactionalAttempt:
    """ 
        Attributes:
            at (datetime.datetime):
            ok (bool):
            provider (str):
            down (bool | Unset):
            error (str | Unset):
            gave_up (bool | Unset):
            temporary (bool | Unset):
     """

    at: datetime.datetime
    ok: bool
    provider: str
    down: bool | Unset = UNSET
    error: str | Unset = UNSET
    gave_up: bool | Unset = UNSET
    temporary: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        ok = self.ok

        provider = self.provider

        down = self.down

        error = self.error

        gave_up = self.gave_up

        temporary = self.temporary


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "ok": ok,
            "provider": provider,
        })
        if down is not UNSET:
            field_dict["down"] = down
        if error is not UNSET:
            field_dict["error"] = error
        if gave_up is not UNSET:
            field_dict["gaveUp"] = gave_up
        if temporary is not UNSET:
            field_dict["temporary"] = temporary

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        ok = d.pop("ok")

        provider = d.pop("provider")

        down = d.pop("down", UNSET)

        error = d.pop("error", UNSET)

        gave_up = d.pop("gaveUp", UNSET)

        temporary = d.pop("temporary", UNSET)

        transactional_attempt = cls(
            at=at,
            ok=ok,
            provider=provider,
            down=down,
            error=error,
            gave_up=gave_up,
            temporary=temporary,
        )

        return transactional_attempt

