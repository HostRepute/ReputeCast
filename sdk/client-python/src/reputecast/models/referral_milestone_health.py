from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_milestone_health_kind import ReferralMilestoneHealthKind
from ..models.referral_milestone_health_status import ReferralMilestoneHealthStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="ReferralMilestoneHealth")



@_attrs_define
class ReferralMilestoneHealth:
    """ 
        Attributes:
            codes_left (int):
            failures (int):
            kind (ReferralMilestoneHealthKind):
            last_status (int):
            status (ReferralMilestoneHealthStatus):
            waiting (int):
            run_out_days (float | Unset):
     """

    codes_left: int
    failures: int
    kind: ReferralMilestoneHealthKind
    last_status: int
    status: ReferralMilestoneHealthStatus
    waiting: int
    run_out_days: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        codes_left = self.codes_left

        failures = self.failures

        kind = self.kind.value

        last_status = self.last_status

        status = self.status.value

        waiting = self.waiting

        run_out_days = self.run_out_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "codesLeft": codes_left,
            "failures": failures,
            "kind": kind,
            "lastStatus": last_status,
            "status": status,
            "waiting": waiting,
        })
        if run_out_days is not UNSET:
            field_dict["runOutDays"] = run_out_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        codes_left = d.pop("codesLeft")

        failures = d.pop("failures")

        kind = ReferralMilestoneHealthKind(d.pop("kind"))




        last_status = d.pop("lastStatus")

        status = ReferralMilestoneHealthStatus(d.pop("status"))




        waiting = d.pop("waiting")

        run_out_days = d.pop("runOutDays", UNSET)

        referral_milestone_health = cls(
            codes_left=codes_left,
            failures=failures,
            kind=kind,
            last_status=last_status,
            status=status,
            waiting=waiting,
            run_out_days=run_out_days,
        )

        return referral_milestone_health

