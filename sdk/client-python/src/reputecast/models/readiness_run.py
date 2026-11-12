from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_run_ran_by_type import ReadinessRunRanByType
from ..models.readiness_run_trigger import ReadinessRunTrigger
from ..models.readiness_run_verdict import ReadinessRunVerdict
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.readiness_check import ReadinessCheck
  from ..models.readiness_counts import ReadinessCounts
  from ..models.readiness_estimates import ReadinessEstimates
  from ..models.readiness_group import ReadinessGroup





T = TypeVar("T", bound="ReadinessRun")



@_attrs_define
class ReadinessRun:
    """ 
        Attributes:
            campaign_id (str):
            checks (list[ReadinessCheck]):
            counts (ReadinessCounts):
            duration_ms (int):
            estimates (ReadinessEstimates):
            groups (list[ReadinessGroup]):
            id (str):
            incomplete (bool):
            ran_at (datetime.datetime):
            trigger (ReadinessRunTrigger):
            verdict (ReadinessRunVerdict):
            version (int):
            problem (str | Unset):
            ran_by (str | Unset):
            ran_by_type (ReadinessRunRanByType | Unset):
            unknown_accepted_at (datetime.datetime | Unset):
            unknown_accepted_by (str | Unset):
     """

    campaign_id: str
    checks: list[ReadinessCheck]
    counts: ReadinessCounts
    duration_ms: int
    estimates: ReadinessEstimates
    groups: list[ReadinessGroup]
    id: str
    incomplete: bool
    ran_at: datetime.datetime
    trigger: ReadinessRunTrigger
    verdict: ReadinessRunVerdict
    version: int
    problem: str | Unset = UNSET
    ran_by: str | Unset = UNSET
    ran_by_type: ReadinessRunRanByType | Unset = UNSET
    unknown_accepted_at: datetime.datetime | Unset = UNSET
    unknown_accepted_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        from ..models.readiness_counts import ReadinessCounts # noqa: PLC0415
        from ..models.readiness_estimates import ReadinessEstimates # noqa: PLC0415
        from ..models.readiness_group import ReadinessGroup # noqa: PLC0415
        campaign_id = self.campaign_id

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        counts = self.counts.to_dict()

        duration_ms = self.duration_ms

        estimates = self.estimates.to_dict()

        groups = []
        for groups_item_data in self.groups:
            groups_item = groups_item_data.to_dict()
            groups.append(groups_item)



        id = self.id

        incomplete = self.incomplete

        ran_at = self.ran_at.isoformat()

        trigger = self.trigger.value

        verdict = self.verdict.value

        version = self.version

        problem = self.problem

        ran_by = self.ran_by

        ran_by_type: str | Unset = UNSET
        if not isinstance(self.ran_by_type, Unset):
            ran_by_type = self.ran_by_type.value


        unknown_accepted_at: str | Unset = UNSET
        if not isinstance(self.unknown_accepted_at, Unset):
            unknown_accepted_at = self.unknown_accepted_at.isoformat()

        unknown_accepted_by = self.unknown_accepted_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "checks": checks,
            "counts": counts,
            "durationMs": duration_ms,
            "estimates": estimates,
            "groups": groups,
            "id": id,
            "incomplete": incomplete,
            "ranAt": ran_at,
            "trigger": trigger,
            "verdict": verdict,
            "version": version,
        })
        if problem is not UNSET:
            field_dict["problem"] = problem
        if ran_by is not UNSET:
            field_dict["ranBy"] = ran_by
        if ran_by_type is not UNSET:
            field_dict["ranByType"] = ran_by_type
        if unknown_accepted_at is not UNSET:
            field_dict["unknownAcceptedAt"] = unknown_accepted_at
        if unknown_accepted_by is not UNSET:
            field_dict["unknownAcceptedBy"] = unknown_accepted_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_check import ReadinessCheck # noqa: PLC0415
        from ..models.readiness_counts import ReadinessCounts # noqa: PLC0415
        from ..models.readiness_estimates import ReadinessEstimates # noqa: PLC0415
        from ..models.readiness_group import ReadinessGroup # noqa: PLC0415
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = ReadinessCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        counts = ReadinessCounts.from_dict(d.pop("counts"))




        duration_ms = d.pop("durationMs")

        estimates = ReadinessEstimates.from_dict(d.pop("estimates"))




        groups = []
        _groups = d.pop("groups")
        for groups_item_data in (_groups):
            groups_item = ReadinessGroup.from_dict(groups_item_data)



            groups.append(groups_item)


        id = d.pop("id")

        incomplete = d.pop("incomplete")

        ran_at = datetime.datetime.fromisoformat(d.pop("ranAt"))




        trigger = ReadinessRunTrigger(d.pop("trigger"))




        verdict = ReadinessRunVerdict(d.pop("verdict"))




        version = d.pop("version")

        problem = d.pop("problem", UNSET)

        ran_by = d.pop("ranBy", UNSET)

        _ran_by_type = d.pop("ranByType", UNSET)
        ran_by_type: ReadinessRunRanByType | Unset
        if isinstance(_ran_by_type,  Unset):
            ran_by_type = UNSET
        else:
            ran_by_type = ReadinessRunRanByType(_ran_by_type)




        _unknown_accepted_at = d.pop("unknownAcceptedAt", UNSET)
        unknown_accepted_at: datetime.datetime | Unset
        if isinstance(_unknown_accepted_at,  Unset):
            unknown_accepted_at = UNSET
        else:
            unknown_accepted_at = datetime.datetime.fromisoformat(_unknown_accepted_at)




        unknown_accepted_by = d.pop("unknownAcceptedBy", UNSET)

        readiness_run = cls(
            campaign_id=campaign_id,
            checks=checks,
            counts=counts,
            duration_ms=duration_ms,
            estimates=estimates,
            groups=groups,
            id=id,
            incomplete=incomplete,
            ran_at=ran_at,
            trigger=trigger,
            verdict=verdict,
            version=version,
            problem=problem,
            ran_by=ran_by,
            ran_by_type=ran_by_type,
            unknown_accepted_at=unknown_accepted_at,
            unknown_accepted_by=unknown_accepted_by,
        )

        return readiness_run

