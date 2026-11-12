from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_variant_checks_key import ComposerVariantChecksKey
from typing import cast

if TYPE_CHECKING:
  from ..models.content_check_issue import ContentCheckIssue





T = TypeVar("T", bound="ComposerVariantChecks")



@_attrs_define
class ComposerVariantChecks:
    """ 
        Attributes:
            blocks (int):
            issues (list[ContentCheckIssue]):
            key (ComposerVariantChecksKey):
            warnings (int):
     """

    blocks: int
    issues: list[ContentCheckIssue]
    key: ComposerVariantChecksKey
    warnings: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.content_check_issue import ContentCheckIssue # noqa: PLC0415
        blocks = self.blocks

        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        key = self.key.value

        warnings = self.warnings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "issues": issues,
            "key": key,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.content_check_issue import ContentCheckIssue # noqa: PLC0415
        d = dict(src_dict)
        blocks = d.pop("blocks")

        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = ContentCheckIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        key = ComposerVariantChecksKey(d.pop("key"))




        warnings = d.pop("warnings")

        composer_variant_checks = cls(
            blocks=blocks,
            issues=issues,
            key=key,
            warnings=warnings,
        )

        return composer_variant_checks

