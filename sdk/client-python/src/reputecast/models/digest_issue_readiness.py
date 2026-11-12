from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="DigestIssueReadiness")



@_attrs_define
class DigestIssueReadiness:
    """ 
        Attributes:
            blocking (list[str]):
            blocks (int):
            passed (int):
            ran_at (datetime.datetime):
            run_id (str):
            verdict (str):
            warnings (int):
     """

    blocking: list[str]
    blocks: int
    passed: int
    ran_at: datetime.datetime
    run_id: str
    verdict: str
    warnings: int





    def to_dict(self) -> dict[str, Any]:
        blocking = self.blocking



        blocks = self.blocks

        passed = self.passed

        ran_at = self.ran_at.isoformat()

        run_id = self.run_id

        verdict = self.verdict

        warnings = self.warnings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocking": blocking,
            "blocks": blocks,
            "passed": passed,
            "ranAt": ran_at,
            "runId": run_id,
            "verdict": verdict,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocking = cast(list[str], d.pop("blocking"))


        blocks = d.pop("blocks")

        passed = d.pop("passed")

        ran_at = datetime.datetime.fromisoformat(d.pop("ranAt"))




        run_id = d.pop("runId")

        verdict = d.pop("verdict")

        warnings = d.pop("warnings")

        digest_issue_readiness = cls(
            blocking=blocking,
            blocks=blocks,
            passed=passed,
            ran_at=ran_at,
            run_id=run_id,
            verdict=verdict,
            warnings=warnings,
        )

        return digest_issue_readiness

