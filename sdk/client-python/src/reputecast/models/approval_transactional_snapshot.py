from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.approval_transactional_test_render import ApprovalTransactionalTestRender





T = TypeVar("T", bound="ApprovalTransactionalSnapshot")



@_attrs_define
class ApprovalTransactionalSnapshot:
    """ 
        Attributes:
            category (str):
            content_summary (str):
            draft_revision (int):
            key (str):
            name (str):
            passed (int):
            template_id (str):
            test_renders (list[ApprovalTransactionalTestRender]):
            variables (list[str]):
            version (int):
            warnings (int):
            breaking (str | Unset):
            live_version (int | Unset):
            published_version (int | Unset):
            subject_after (str | Unset):
            subject_before (str | Unset):
     """

    category: str
    content_summary: str
    draft_revision: int
    key: str
    name: str
    passed: int
    template_id: str
    test_renders: list[ApprovalTransactionalTestRender]
    variables: list[str]
    version: int
    warnings: int
    breaking: str | Unset = UNSET
    live_version: int | Unset = UNSET
    published_version: int | Unset = UNSET
    subject_after: str | Unset = UNSET
    subject_before: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_transactional_test_render import ApprovalTransactionalTestRender # noqa: PLC0415
        category = self.category

        content_summary = self.content_summary

        draft_revision = self.draft_revision

        key = self.key

        name = self.name

        passed = self.passed

        template_id = self.template_id

        test_renders = []
        for test_renders_item_data in self.test_renders:
            test_renders_item = test_renders_item_data.to_dict()
            test_renders.append(test_renders_item)



        variables = self.variables



        version = self.version

        warnings = self.warnings

        breaking = self.breaking

        live_version = self.live_version

        published_version = self.published_version

        subject_after = self.subject_after

        subject_before = self.subject_before


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "category": category,
            "contentSummary": content_summary,
            "draftRevision": draft_revision,
            "key": key,
            "name": name,
            "passed": passed,
            "templateId": template_id,
            "testRenders": test_renders,
            "variables": variables,
            "version": version,
            "warnings": warnings,
        })
        if breaking is not UNSET:
            field_dict["breaking"] = breaking
        if live_version is not UNSET:
            field_dict["liveVersion"] = live_version
        if published_version is not UNSET:
            field_dict["published_version"] = published_version
        if subject_after is not UNSET:
            field_dict["subjectAfter"] = subject_after
        if subject_before is not UNSET:
            field_dict["subjectBefore"] = subject_before

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_transactional_test_render import ApprovalTransactionalTestRender # noqa: PLC0415
        d = dict(src_dict)
        category = d.pop("category")

        content_summary = d.pop("contentSummary")

        draft_revision = d.pop("draftRevision")

        key = d.pop("key")

        name = d.pop("name")

        passed = d.pop("passed")

        template_id = d.pop("templateId")

        test_renders = []
        _test_renders = d.pop("testRenders")
        for test_renders_item_data in (_test_renders):
            test_renders_item = ApprovalTransactionalTestRender.from_dict(test_renders_item_data)



            test_renders.append(test_renders_item)


        variables = cast(list[str], d.pop("variables"))


        version = d.pop("version")

        warnings = d.pop("warnings")

        breaking = d.pop("breaking", UNSET)

        live_version = d.pop("liveVersion", UNSET)

        published_version = d.pop("published_version", UNSET)

        subject_after = d.pop("subjectAfter", UNSET)

        subject_before = d.pop("subjectBefore", UNSET)

        approval_transactional_snapshot = cls(
            category=category,
            content_summary=content_summary,
            draft_revision=draft_revision,
            key=key,
            name=name,
            passed=passed,
            template_id=template_id,
            test_renders=test_renders,
            variables=variables,
            version=version,
            warnings=warnings,
            breaking=breaking,
            live_version=live_version,
            published_version=published_version,
            subject_after=subject_after,
            subject_before=subject_before,
        )

        return approval_transactional_snapshot

