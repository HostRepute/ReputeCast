from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_sync_step_status import EarnSyncStepStatus
from ..models.earn_sync_step_step import EarnSyncStepStep
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnSyncStep")



@_attrs_define
class EarnSyncStep:
    """ 
        Attributes:
            status (EarnSyncStepStatus):
            step (EarnSyncStepStep):
            amount_cents (int | Unset):
            at (datetime.datetime | Unset):
            error (str | Unset):
            id (str | Unset):
            tries (int | Unset):
     """

    status: EarnSyncStepStatus
    step: EarnSyncStepStep
    amount_cents: int | Unset = UNSET
    at: datetime.datetime | Unset = UNSET
    error: str | Unset = UNSET
    id: str | Unset = UNSET
    tries: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        step = self.step.value

        amount_cents = self.amount_cents

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        error = self.error

        id = self.id

        tries = self.tries


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
            "step": step,
        })
        if amount_cents is not UNSET:
            field_dict["amountCents"] = amount_cents
        if at is not UNSET:
            field_dict["at"] = at
        if error is not UNSET:
            field_dict["error"] = error
        if id is not UNSET:
            field_dict["id"] = id
        if tries is not UNSET:
            field_dict["tries"] = tries

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = EarnSyncStepStatus(d.pop("status"))




        step = EarnSyncStepStep(d.pop("step"))




        amount_cents = d.pop("amountCents", UNSET)

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        error = d.pop("error", UNSET)

        id = d.pop("id", UNSET)

        tries = d.pop("tries", UNSET)

        earn_sync_step = cls(
            status=status,
            step=step,
            amount_cents=amount_cents,
            at=at,
            error=error,
            id=id,
            tries=tries,
        )

        return earn_sync_step

