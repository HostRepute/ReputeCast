from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_checks_verdict import ComposerChecksVerdict
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.composer_dismissed_issue import ComposerDismissedIssue
  from ..models.composer_variant_checks import ComposerVariantChecks
  from ..models.content_check_image import ContentCheckImage
  from ..models.content_check_issue import ContentCheckIssue
  from ..models.content_check_link import ContentCheckLink
  from ..models.content_check_section import ContentCheckSection
  from ..models.content_check_unknown_domain import ContentCheckUnknownDomain
  from ..models.mjml_problem import MJMLProblem





T = TypeVar("T", bound="ComposerChecks")



@_attrs_define
class ComposerChecks:
    """ 
        Attributes:
            blocks (int):
            body_heuristics (list[str]):
            checked_at (datetime.datetime):
            clip_bytes (int):
            dismissed (list[ComposerDismissedIssue]):
            image_list (list[ContentCheckImage]):
            image_share (float):
            images (int):
            images_insecure (int):
            images_missing_alt (int):
            issue_keys (list[str]):
            issues (list[ContentCheckIssue]):
            link_domains (int):
            link_domains_unknown (list[ContentCheckUnknownDomain]):
            link_list (list[ContentCheckLink]):
            links (int):
            links_blocklisted (int):
            links_broken (int):
            lint (list[MJMLProblem]):
            plain_text (str):
            preheader_heuristics (list[str]):
            sections (list[ContentCheckSection]):
            size_bytes (int):
            subject_heuristics (list[str]):
            uri_lists (list[str]):
            variants (list[ComposerVariantChecks]):
            verdict (ComposerChecksVerdict):
            warnings (int):
            words (int):
            footer_removed_at (datetime.datetime | Unset):
     """

    blocks: int
    body_heuristics: list[str]
    checked_at: datetime.datetime
    clip_bytes: int
    dismissed: list[ComposerDismissedIssue]
    image_list: list[ContentCheckImage]
    image_share: float
    images: int
    images_insecure: int
    images_missing_alt: int
    issue_keys: list[str]
    issues: list[ContentCheckIssue]
    link_domains: int
    link_domains_unknown: list[ContentCheckUnknownDomain]
    link_list: list[ContentCheckLink]
    links: int
    links_blocklisted: int
    links_broken: int
    lint: list[MJMLProblem]
    plain_text: str
    preheader_heuristics: list[str]
    sections: list[ContentCheckSection]
    size_bytes: int
    subject_heuristics: list[str]
    uri_lists: list[str]
    variants: list[ComposerVariantChecks]
    verdict: ComposerChecksVerdict
    warnings: int
    words: int
    footer_removed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_dismissed_issue import ComposerDismissedIssue # noqa: PLC0415
        from ..models.composer_variant_checks import ComposerVariantChecks # noqa: PLC0415
        from ..models.content_check_image import ContentCheckImage # noqa: PLC0415
        from ..models.content_check_issue import ContentCheckIssue # noqa: PLC0415
        from ..models.content_check_link import ContentCheckLink # noqa: PLC0415
        from ..models.content_check_section import ContentCheckSection # noqa: PLC0415
        from ..models.content_check_unknown_domain import ContentCheckUnknownDomain # noqa: PLC0415
        from ..models.mjml_problem import MJMLProblem # noqa: PLC0415
        blocks = self.blocks

        body_heuristics = self.body_heuristics



        checked_at = self.checked_at.isoformat()

        clip_bytes = self.clip_bytes

        dismissed = []
        for dismissed_item_data in self.dismissed:
            dismissed_item = dismissed_item_data.to_dict()
            dismissed.append(dismissed_item)



        image_list = []
        for image_list_item_data in self.image_list:
            image_list_item = image_list_item_data.to_dict()
            image_list.append(image_list_item)



        image_share = self.image_share

        images = self.images

        images_insecure = self.images_insecure

        images_missing_alt = self.images_missing_alt

        issue_keys = self.issue_keys



        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        link_domains = self.link_domains

        link_domains_unknown = []
        for link_domains_unknown_item_data in self.link_domains_unknown:
            link_domains_unknown_item = link_domains_unknown_item_data.to_dict()
            link_domains_unknown.append(link_domains_unknown_item)



        link_list = []
        for link_list_item_data in self.link_list:
            link_list_item = link_list_item_data.to_dict()
            link_list.append(link_list_item)



        links = self.links

        links_blocklisted = self.links_blocklisted

        links_broken = self.links_broken

        lint = []
        for lint_item_data in self.lint:
            lint_item = lint_item_data.to_dict()
            lint.append(lint_item)



        plain_text = self.plain_text

        preheader_heuristics = self.preheader_heuristics



        sections = []
        for sections_item_data in self.sections:
            sections_item = sections_item_data.to_dict()
            sections.append(sections_item)



        size_bytes = self.size_bytes

        subject_heuristics = self.subject_heuristics



        uri_lists = self.uri_lists



        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)



        verdict = self.verdict.value

        warnings = self.warnings

        words = self.words

        footer_removed_at: str | Unset = UNSET
        if not isinstance(self.footer_removed_at, Unset):
            footer_removed_at = self.footer_removed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "bodyHeuristics": body_heuristics,
            "checkedAt": checked_at,
            "clipBytes": clip_bytes,
            "dismissed": dismissed,
            "imageList": image_list,
            "imageShare": image_share,
            "images": images,
            "imagesInsecure": images_insecure,
            "imagesMissingAlt": images_missing_alt,
            "issueKeys": issue_keys,
            "issues": issues,
            "linkDomains": link_domains,
            "linkDomainsUnknown": link_domains_unknown,
            "linkList": link_list,
            "links": links,
            "linksBlocklisted": links_blocklisted,
            "linksBroken": links_broken,
            "lint": lint,
            "plainText": plain_text,
            "preheaderHeuristics": preheader_heuristics,
            "sections": sections,
            "sizeBytes": size_bytes,
            "subjectHeuristics": subject_heuristics,
            "uriLists": uri_lists,
            "variants": variants,
            "verdict": verdict,
            "warnings": warnings,
            "words": words,
        })
        if footer_removed_at is not UNSET:
            field_dict["footerRemovedAt"] = footer_removed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_dismissed_issue import ComposerDismissedIssue # noqa: PLC0415
        from ..models.composer_variant_checks import ComposerVariantChecks # noqa: PLC0415
        from ..models.content_check_image import ContentCheckImage # noqa: PLC0415
        from ..models.content_check_issue import ContentCheckIssue # noqa: PLC0415
        from ..models.content_check_link import ContentCheckLink # noqa: PLC0415
        from ..models.content_check_section import ContentCheckSection # noqa: PLC0415
        from ..models.content_check_unknown_domain import ContentCheckUnknownDomain # noqa: PLC0415
        from ..models.mjml_problem import MJMLProblem # noqa: PLC0415
        d = dict(src_dict)
        blocks = d.pop("blocks")

        body_heuristics = cast(list[str], d.pop("bodyHeuristics"))


        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        clip_bytes = d.pop("clipBytes")

        dismissed = []
        _dismissed = d.pop("dismissed")
        for dismissed_item_data in (_dismissed):
            dismissed_item = ComposerDismissedIssue.from_dict(dismissed_item_data)



            dismissed.append(dismissed_item)


        image_list = []
        _image_list = d.pop("imageList")
        for image_list_item_data in (_image_list):
            image_list_item = ContentCheckImage.from_dict(image_list_item_data)



            image_list.append(image_list_item)


        image_share = d.pop("imageShare")

        images = d.pop("images")

        images_insecure = d.pop("imagesInsecure")

        images_missing_alt = d.pop("imagesMissingAlt")

        issue_keys = cast(list[str], d.pop("issueKeys"))


        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = ContentCheckIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        link_domains = d.pop("linkDomains")

        link_domains_unknown = []
        _link_domains_unknown = d.pop("linkDomainsUnknown")
        for link_domains_unknown_item_data in (_link_domains_unknown):
            link_domains_unknown_item = ContentCheckUnknownDomain.from_dict(link_domains_unknown_item_data)



            link_domains_unknown.append(link_domains_unknown_item)


        link_list = []
        _link_list = d.pop("linkList")
        for link_list_item_data in (_link_list):
            link_list_item = ContentCheckLink.from_dict(link_list_item_data)



            link_list.append(link_list_item)


        links = d.pop("links")

        links_blocklisted = d.pop("linksBlocklisted")

        links_broken = d.pop("linksBroken")

        lint = []
        _lint = d.pop("lint")
        for lint_item_data in (_lint):
            lint_item = MJMLProblem.from_dict(lint_item_data)



            lint.append(lint_item)


        plain_text = d.pop("plainText")

        preheader_heuristics = cast(list[str], d.pop("preheaderHeuristics"))


        sections = []
        _sections = d.pop("sections")
        for sections_item_data in (_sections):
            sections_item = ContentCheckSection.from_dict(sections_item_data)



            sections.append(sections_item)


        size_bytes = d.pop("sizeBytes")

        subject_heuristics = cast(list[str], d.pop("subjectHeuristics"))


        uri_lists = cast(list[str], d.pop("uriLists"))


        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = ComposerVariantChecks.from_dict(variants_item_data)



            variants.append(variants_item)


        verdict = ComposerChecksVerdict(d.pop("verdict"))




        warnings = d.pop("warnings")

        words = d.pop("words")

        _footer_removed_at = d.pop("footerRemovedAt", UNSET)
        footer_removed_at: datetime.datetime | Unset
        if isinstance(_footer_removed_at,  Unset):
            footer_removed_at = UNSET
        else:
            footer_removed_at = datetime.datetime.fromisoformat(_footer_removed_at)




        composer_checks = cls(
            blocks=blocks,
            body_heuristics=body_heuristics,
            checked_at=checked_at,
            clip_bytes=clip_bytes,
            dismissed=dismissed,
            image_list=image_list,
            image_share=image_share,
            images=images,
            images_insecure=images_insecure,
            images_missing_alt=images_missing_alt,
            issue_keys=issue_keys,
            issues=issues,
            link_domains=link_domains,
            link_domains_unknown=link_domains_unknown,
            link_list=link_list,
            links=links,
            links_blocklisted=links_blocklisted,
            links_broken=links_broken,
            lint=lint,
            plain_text=plain_text,
            preheader_heuristics=preheader_heuristics,
            sections=sections,
            size_bytes=size_bytes,
            subject_heuristics=subject_heuristics,
            uri_lists=uri_lists,
            variants=variants,
            verdict=verdict,
            warnings=warnings,
            words=words,
            footer_removed_at=footer_removed_at,
        )

        return composer_checks

