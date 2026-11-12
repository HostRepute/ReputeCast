from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_checks_verdict import TransactionalChecksVerdict
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_check import TransactionalCheck
  from ..models.transactional_test_render import TransactionalTestRender
  from ..models.transactional_variable_report import TransactionalVariableReport





T = TypeVar("T", bound="TransactionalChecks")



@_attrs_define
class TransactionalChecks:
    """ 
        Attributes:
            blocks (int):
            checks (list[TransactionalCheck]):
            clip_bytes (int):
            images (int):
            links (int):
            passed (int):
            plain_text (str):
            size_bytes (int):
            test_data (list[TransactionalTestRender]):
            variables (TransactionalVariableReport):
            verdict (TransactionalChecksVerdict):
            warnings (int):
     """

    blocks: int
    checks: list[TransactionalCheck]
    clip_bytes: int
    images: int
    links: int
    passed: int
    plain_text: str
    size_bytes: int
    test_data: list[TransactionalTestRender]
    variables: TransactionalVariableReport
    verdict: TransactionalChecksVerdict
    warnings: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_check import TransactionalCheck # noqa: PLC0415
        from ..models.transactional_test_render import TransactionalTestRender # noqa: PLC0415
        from ..models.transactional_variable_report import TransactionalVariableReport # noqa: PLC0415
        blocks = self.blocks

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        clip_bytes = self.clip_bytes

        images = self.images

        links = self.links

        passed = self.passed

        plain_text = self.plain_text

        size_bytes = self.size_bytes

        test_data = []
        for test_data_item_data in self.test_data:
            test_data_item = test_data_item_data.to_dict()
            test_data.append(test_data_item)



        variables = self.variables.to_dict()

        verdict = self.verdict.value

        warnings = self.warnings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "checks": checks,
            "clipBytes": clip_bytes,
            "images": images,
            "links": links,
            "passed": passed,
            "plainText": plain_text,
            "sizeBytes": size_bytes,
            "testData": test_data,
            "variables": variables,
            "verdict": verdict,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_check import TransactionalCheck # noqa: PLC0415
        from ..models.transactional_test_render import TransactionalTestRender # noqa: PLC0415
        from ..models.transactional_variable_report import TransactionalVariableReport # noqa: PLC0415
        d = dict(src_dict)
        blocks = d.pop("blocks")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = TransactionalCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        clip_bytes = d.pop("clipBytes")

        images = d.pop("images")

        links = d.pop("links")

        passed = d.pop("passed")

        plain_text = d.pop("plainText")

        size_bytes = d.pop("sizeBytes")

        test_data = []
        _test_data = d.pop("testData")
        for test_data_item_data in (_test_data):
            test_data_item = TransactionalTestRender.from_dict(test_data_item_data)



            test_data.append(test_data_item)


        variables = TransactionalVariableReport.from_dict(d.pop("variables"))




        verdict = TransactionalChecksVerdict(d.pop("verdict"))




        warnings = d.pop("warnings")

        transactional_checks = cls(
            blocks=blocks,
            checks=checks,
            clip_bytes=clip_bytes,
            images=images,
            links=links,
            passed=passed,
            plain_text=plain_text,
            size_bytes=size_bytes,
            test_data=test_data,
            variables=variables,
            verdict=verdict,
            warnings=warnings,
        )

        return transactional_checks

