from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="PlacementAuthReadiness")



@_attrs_define
class PlacementAuthReadiness:
    """ 
        Attributes:
            blocking (list[str]): The blocking checks, in the reader's language
            blocks (int):
            evidence (list[str]): One sentence per blocking check
            passed (int):
            warnings (int):
     """

    blocking: list[str]
    blocks: int
    evidence: list[str]
    passed: int
    warnings: int





    def to_dict(self) -> dict[str, Any]:
        blocking = self.blocking



        blocks = self.blocks

        evidence = self.evidence



        passed = self.passed

        warnings = self.warnings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocking": blocking,
            "blocks": blocks,
            "evidence": evidence,
            "passed": passed,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocking = cast(list[str], d.pop("blocking"))


        blocks = d.pop("blocks")

        evidence = cast(list[str], d.pop("evidence"))


        passed = d.pop("passed")

        warnings = d.pop("warnings")

        placement_auth_readiness = cls(
            blocking=blocking,
            blocks=blocks,
            evidence=evidence,
            passed=passed,
            warnings=warnings,
        )

        return placement_auth_readiness

