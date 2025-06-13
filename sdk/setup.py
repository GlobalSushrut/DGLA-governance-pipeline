from setuptools import setup, find_packages

setup(
    name="dgla-sdk",
    version="1.0.0",
    packages=find_packages(),
    install_requires=[
        "requests>=2.28.0",
        "pyjwt>=2.6.0",
        "cryptography>=39.0.0",
    ],
    author="DGLA Team",
    author_email="info@dgla.io",
    description="Client SDK for DGLA infrastructure",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    url="https://github.com/umesh/dgla",
    classifiers=[
        "Development Status :: 5 - Production/Stable",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
    ],
    python_requires=">=3.7",
)
